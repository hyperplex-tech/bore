package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AddTunnel adds a tunnel to a group in the config file. If the group doesn't
// exist, it is created. The file is rewritten atomically.
func AddTunnel(path string, groupName string, tunnel TunnelConfig) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}

	// Check for duplicate name.
	for _, rt := range cfg.AllTunnels() {
		if rt.Name == tunnel.Name {
			return fmt.Errorf("tunnel %q already exists in group %q", tunnel.Name, rt.Group)
		}
	}

	g := cfg.Groups[groupName]
	if g.Description == "" && groupName != "" {
		g.Description = groupName
	}
	g.Tunnels = append(g.Tunnels, tunnel)
	cfg.Groups[groupName] = g

	return Save(path, cfg)
}

// RemoveTunnel removes a tunnel by name from the config file.
func RemoveTunnel(path string, tunnelName string) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}

	for groupName, group := range cfg.Groups {
		for i, t := range group.Tunnels {
			if t.Name == tunnelName {
				group.Tunnels = append(group.Tunnels[:i], group.Tunnels[i+1:]...)
				cfg.Groups[groupName] = group
				return Save(path, cfg)
			}
		}
	}
	return fmt.Errorf("tunnel %q not found", tunnelName)
}

// UpdateTunnel replaces a tunnel's config, optionally moving it to a new group.
func UpdateTunnel(path string, originalName string, updated TunnelConfig, newGroup string) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}

	// Find and remove the tunnel from its current group.
	found := false
	for groupName, group := range cfg.Groups {
		for i, t := range group.Tunnels {
			if t.Name == originalName {
				group.Tunnels = append(group.Tunnels[:i], group.Tunnels[i+1:]...)
				cfg.Groups[groupName] = group
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("tunnel %q not found", originalName)
	}

	// Check for name conflicts if renamed.
	if updated.Name != originalName {
		for _, rt := range cfg.AllTunnels() {
			if rt.Name == updated.Name {
				return fmt.Errorf("tunnel %q already exists in group %q", updated.Name, rt.Group)
			}
		}
	}

	// Add to the target group.
	g := cfg.Groups[newGroup]
	if g.Description == "" && newGroup != "" {
		g.Description = newGroup
	}
	g.Tunnels = append(g.Tunnels, updated)
	cfg.Groups[newGroup] = g

	return Save(path, cfg)
}

// DuplicateTunnel copies a tunnel and adds it with " - Copy" appended to the name.
func DuplicateTunnel(path string, name string) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}

	rt, found := cfg.FindTunnel(name)
	if !found {
		return fmt.Errorf("tunnel %q not found", name)
	}

	dup := rt.TunnelConfig
	dup.Name = name + " - Copy"

	return AddTunnel(path, rt.Group, dup)
}

// AddGroup creates a new empty group in the config.
func AddGroup(path string, name string, description string) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}
	if _, exists := cfg.Groups[name]; exists {
		return fmt.Errorf("group %q already exists", name)
	}
	if description == "" {
		description = name
	}
	cfg.Groups[name] = Group{Description: description}
	return Save(path, cfg)
}

// RemoveGroup removes an empty group from the config.
func RemoveGroup(path string, groupName string) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}
	g, exists := cfg.Groups[groupName]
	if !exists {
		return fmt.Errorf("group %q not found", groupName)
	}
	if len(g.Tunnels) > 0 {
		return fmt.Errorf("group %q has %d tunnels — remove them first", groupName, len(g.Tunnels))
	}
	delete(cfg.Groups, groupName)
	return Save(path, cfg)
}

// RenameGroup renames a group, moving all its tunnels to the new name.
func RenameGroup(path string, oldName string, newName string) error {
	cfg, err := LoadOrDefault(path)
	if err != nil {
		return err
	}
	g, exists := cfg.Groups[oldName]
	if !exists {
		return fmt.Errorf("group %q not found", oldName)
	}
	if _, exists := cfg.Groups[newName]; exists {
		return fmt.Errorf("group %q already exists", newName)
	}
	if newName == "" {
		return fmt.Errorf("new group name cannot be empty")
	}
	if g.Description == oldName {
		g.Description = newName
	}
	cfg.Groups[newName] = g
	delete(cfg.Groups, oldName)
	return Save(path, cfg)
}

// Save writes the config to the YAML file atomically.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Write to a temp file, then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming config: %w", err)
	}
	return nil
}
