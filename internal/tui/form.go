package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
)

// formField indices
const (
	fieldName = iota
	fieldGroup
	fieldType
	fieldLocalHost
	fieldLocalPort
	fieldRemoteHost
	fieldRemotePort
	fieldSSHHost
	fieldSSHPort
	fieldSSHUser
	fieldAuthMethod
	fieldIdentityFile
	fieldJumpHosts
	fieldK8sContext
	fieldK8sNamespace
	fieldK8sResource
	fieldPreConnect
	fieldPostConnect
	fieldCount
)

type tunnelForm struct {
	visible  bool
	editing  bool // true = edit mode, false = add mode
	original string // original tunnel name when editing

	inputs []textinput.Model
	focus  int
	width  int
	height int
	scroll int // scroll offset for long form
	err    string
}

func newTunnelForm() tunnelForm {
	inputs := make([]textinput.Model, fieldCount)

	placeholders := [fieldCount]string{
		"tunnel-name",
		"default",
		"local|remote|dynamic|k8s",
		"127.0.0.1",
		"8080",
		"remote.example.com",
		"80",
		"bastion.example.com",
		"22",
		"user",
		"agent|key",
		"/path/to/key",
		"jump1,jump2",
		"my-context",
		"default",
		"svc/my-service",
		"echo pre-connect",
		"echo post-connect",
	}

	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.SetWidth(35)
		s := ti.Styles()
		s.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
		s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
		s.Blurred.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa"))
		s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		ti.SetStyles(s)
		inputs[i] = ti
	}

	return tunnelForm{inputs: inputs}
}

func (f *tunnelForm) showAdd() tea.Cmd {
	f.visible = true
	f.editing = false
	f.original = ""
	f.err = ""
	f.focus = 0
	f.scroll = 0
	for i := range f.inputs {
		f.inputs[i].SetValue("")
		f.inputs[i].Blur()
	}
	return f.inputs[f.focus].Focus()
}

func (f *tunnelForm) showEdit(t *borev1.Tunnel, tc *internalconfig.TunnelConfig) tea.Cmd {
	f.visible = true
	f.editing = true
	f.original = t.Name
	f.err = ""
	f.focus = 0
	f.scroll = 0

	for i := range f.inputs {
		f.inputs[i].SetValue("")
		f.inputs[i].Blur()
	}

	f.inputs[fieldName].SetValue(t.Name)
	f.inputs[fieldGroup].SetValue(t.Group)

	tunnelType := "local"
	switch t.Type {
	case borev1.TunnelType_TUNNEL_TYPE_REMOTE:
		tunnelType = "remote"
	case borev1.TunnelType_TUNNEL_TYPE_DYNAMIC:
		tunnelType = "dynamic"
	case borev1.TunnelType_TUNNEL_TYPE_K8S:
		tunnelType = "k8s"
	}
	f.inputs[fieldType].SetValue(tunnelType)
	f.inputs[fieldLocalHost].SetValue(t.LocalHost)
	if t.LocalPort > 0 {
		f.inputs[fieldLocalPort].SetValue(strconv.Itoa(int(t.LocalPort)))
	}
	f.inputs[fieldRemoteHost].SetValue(t.RemoteHost)
	if t.RemotePort > 0 {
		f.inputs[fieldRemotePort].SetValue(strconv.Itoa(int(t.RemotePort)))
	}
	f.inputs[fieldSSHHost].SetValue(t.SshHost)
	if t.SshPort > 0 {
		f.inputs[fieldSSHPort].SetValue(strconv.Itoa(int(t.SshPort)))
	}
	f.inputs[fieldSSHUser].SetValue(t.SshUser)
	f.inputs[fieldAuthMethod].SetValue(t.AuthMethod)
	f.inputs[fieldIdentityFile].SetValue(t.IdentityFile)
	if len(t.JumpHosts) > 0 {
		f.inputs[fieldJumpHosts].SetValue(strings.Join(t.JumpHosts, ","))
	}
	f.inputs[fieldK8sContext].SetValue(t.K8SContext)
	f.inputs[fieldK8sNamespace].SetValue(t.K8SNamespace)
	f.inputs[fieldK8sResource].SetValue(t.K8SResource)

	if tc != nil && tc.Hooks != nil {
		f.inputs[fieldPreConnect].SetValue(tc.Hooks.PreConnect)
		f.inputs[fieldPostConnect].SetValue(tc.Hooks.PostConnect)
	}

	return f.inputs[f.focus].Focus()
}

func (f *tunnelForm) hide() {
	f.visible = false
	f.err = ""
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
}

func (f *tunnelForm) focusNext() tea.Cmd {
	f.inputs[f.focus].Blur()
	f.focus = (f.focus + 1) % fieldCount
	return f.inputs[f.focus].Focus()
}

func (f *tunnelForm) focusPrev() tea.Cmd {
	f.inputs[f.focus].Blur()
	f.focus = (f.focus - 1 + fieldCount) % fieldCount
	return f.inputs[f.focus].Focus()
}

func (f *tunnelForm) update(msg tea.Msg) (bool, tea.Cmd) {
	if !f.visible {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			f.hide()
			return true, nil
		case "tab", "down":
			return true, f.focusNext()
		case "shift+tab", "up":
			return true, f.focusPrev()
		case "ctrl+s":
			// Submit
			return true, nil
		}
	}

	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return true, cmd
}

// build creates a TunnelConfig from the form inputs, returning an error string if validation fails.
func (f *tunnelForm) build() (internalconfig.TunnelConfig, string, string) {
	name := strings.TrimSpace(f.inputs[fieldName].Value())
	group := strings.TrimSpace(f.inputs[fieldGroup].Value())
	tunnelType := strings.TrimSpace(f.inputs[fieldType].Value())
	localPort := strings.TrimSpace(f.inputs[fieldLocalPort].Value())
	remoteHost := strings.TrimSpace(f.inputs[fieldRemoteHost].Value())
	remotePort := strings.TrimSpace(f.inputs[fieldRemotePort].Value())
	sshHost := strings.TrimSpace(f.inputs[fieldSSHHost].Value())

	if name == "" {
		return internalconfig.TunnelConfig{}, "", "Name is required"
	}
	if group == "" {
		group = "default"
	}
	if tunnelType == "" {
		tunnelType = "local"
	}

	switch tunnelType {
	case "local", "remote", "dynamic", "k8s":
	default:
		return internalconfig.TunnelConfig{}, "", "Invalid type (local, remote, dynamic, k8s)"
	}

	lp, err := strconv.Atoi(localPort)
	if err != nil || lp <= 0 {
		return internalconfig.TunnelConfig{}, "", "Local port must be a positive number"
	}

	tc := internalconfig.TunnelConfig{
		Name:      name,
		Type:      tunnelType,
		LocalHost: strings.TrimSpace(f.inputs[fieldLocalHost].Value()),
		LocalPort: lp,
	}

	switch tunnelType {
	case "k8s":
		k8sResource := strings.TrimSpace(f.inputs[fieldK8sResource].Value())
		rp, _ := strconv.Atoi(remotePort)
		if k8sResource == "" || rp <= 0 {
			return internalconfig.TunnelConfig{}, "", "K8s tunnels require resource and remote port"
		}
		tc.RemotePort = rp
		tc.K8sContext = strings.TrimSpace(f.inputs[fieldK8sContext].Value())
		tc.K8sNamespace = strings.TrimSpace(f.inputs[fieldK8sNamespace].Value())
		tc.K8sResource = k8sResource
	case "dynamic":
		if sshHost == "" {
			return internalconfig.TunnelConfig{}, "", "SOCKS5 tunnels require SSH host"
		}
		tc.SSHHost = sshHost
	default:
		rp, _ := strconv.Atoi(remotePort)
		if remoteHost == "" || rp <= 0 || sshHost == "" {
			return internalconfig.TunnelConfig{}, "", "Requires remote host, remote port, and SSH host"
		}
		tc.RemoteHost = remoteHost
		tc.RemotePort = rp
		tc.SSHHost = sshHost
	}

	if sp := strings.TrimSpace(f.inputs[fieldSSHPort].Value()); sp != "" {
		port, _ := strconv.Atoi(sp)
		tc.SSHPort = port
	}
	tc.SSHUser = strings.TrimSpace(f.inputs[fieldSSHUser].Value())
	tc.AuthMethod = strings.TrimSpace(f.inputs[fieldAuthMethod].Value())
	tc.IdentityFile = strings.TrimSpace(f.inputs[fieldIdentityFile].Value())

	if jh := strings.TrimSpace(f.inputs[fieldJumpHosts].Value()); jh != "" {
		tc.JumpHosts = strings.Split(jh, ",")
	}

	pre := strings.TrimSpace(f.inputs[fieldPreConnect].Value())
	post := strings.TrimSpace(f.inputs[fieldPostConnect].Value())
	if pre != "" || post != "" {
		tc.Hooks = &internalconfig.Hooks{
			PreConnect:  pre,
			PostConnect: post,
		}
	}

	return tc, group, ""
}

func (f *tunnelForm) view(s styles) string {
	if !f.visible {
		return ""
	}

	title := "Add Tunnel"
	if f.editing {
		title = "Edit Tunnel"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#4a9eff"))

	labels := [fieldCount]string{
		"Name:", "Group:", "Type:", "Local Host:", "Local Port:",
		"Remote Host:", "Remote Port:", "SSH Host:", "SSH Port:",
		"SSH User:", "Auth Method:", "Identity File:", "Jump Hosts:",
		"K8s Context:", "K8s Namespace:", "K8s Resource:",
		"Pre-Connect:", "Post-Connect:",
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Width(16).
		Align(lipgloss.Right)

	focusLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4a9eff")).
		Width(16).
		Align(lipgloss.Right).
		Bold(true)

	// Calculate visible area
	maxVisible := f.height - 6 // title, error, hints, borders
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Ensure focused field is visible
	if f.focus < f.scroll {
		f.scroll = f.focus
	}
	if f.focus >= f.scroll+maxVisible {
		f.scroll = f.focus - maxVisible + 1
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	end := f.scroll + maxVisible
	if end > fieldCount {
		end = fieldCount
	}

	for i := f.scroll; i < end; i++ {
		ls := labelStyle
		if i == f.focus {
			ls = focusLabel
		}
		b.WriteString(ls.Render(labels[i]) + " " + f.inputs[i].View() + "\n")
	}

	if f.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		b.WriteString("\n" + errStyle.Render(f.err))
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	b.WriteString("\n" + hint.Render("Tab/↓ next  Shift+Tab/↑ prev  Ctrl+S save  Esc cancel"))

	boxWidth := min(65, f.width-4)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4a9eff")).
		Padding(1, 2).
		Width(boxWidth).
		Background(lipgloss.Color("#1a1a2e"))

	return boxStyle.Render(b.String())
}
