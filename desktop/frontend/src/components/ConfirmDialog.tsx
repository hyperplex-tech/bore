import React from "react";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  danger = false,
  onConfirm,
  onCancel,
}) => {
  if (!open) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-bore-surface border border-bore-border rounded-lg w-[340px] p-4">
        <h2 className="text-sm font-bold text-bore-text mb-2">{title}</h2>
        <p className="text-xs text-bore-text-muted mb-4">{message}</p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-xs border border-bore-border text-bore-text-muted rounded hover:text-bore-text transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className={`px-3 py-1.5 text-xs text-white rounded transition-colors ${
              danger
                ? "bg-bore-error hover:bg-bore-error/80"
                : "bg-bore-accent hover:bg-bore-accent-hover"
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
};
