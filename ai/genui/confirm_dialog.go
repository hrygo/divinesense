package genui

// ConfirmDialogData represents data for a confirmation dialog.
type ConfirmDialogData struct {
	Title       string `json:"title"`
	Message     string `json:"message"`
	ConfirmText string `json:"confirm_text"`
	CancelText  string `json:"cancel_text"`
	Danger      bool   `json:"danger"` // Red button for dangerous actions
}

// NewConfirmDialog creates a new confirmation dialog component.
func NewConfirmDialog(title, message string, payload any, danger bool) *UIComponent {
	confirmStyle := "primary"
	if danger {
		confirmStyle = "danger"
	}

	return &UIComponent{
		Type: ComponentConfirmDialog,
		ID:   generateID(),
		Data: &ConfirmDialogData{
			Title:       title,
			Message:     message,
			ConfirmText: "确认",
			CancelText:  "取消",
			Danger:      danger,
		},
		Actions: []UIAction{
			{
				ID:      "confirm",
				Type:    "button",
				Label:   "确认",
				Style:   confirmStyle,
				Payload: payload,
			},
			{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "secondary",
			},
		},
	}
}

// NewConfirmDialogWithLabels creates a confirmation dialog with custom button labels.
func NewConfirmDialogWithLabels(title, message, confirmLabel, cancelLabel string, payload any, danger bool) *UIComponent {
	confirmStyle := "primary"
	if danger {
		confirmStyle = "danger"
	}

	return &UIComponent{
		Type: ComponentConfirmDialog,
		ID:   generateID(),
		Data: &ConfirmDialogData{
			Title:       title,
			Message:     message,
			ConfirmText: confirmLabel,
			CancelText:  cancelLabel,
			Danger:      danger,
		},
		Actions: []UIAction{
			{
				ID:      "confirm",
				Type:    "button",
				Label:   confirmLabel,
				Style:   confirmStyle,
				Payload: payload,
			},
			{
				ID:    "cancel",
				Type:  "button",
				Label: cancelLabel,
				Style: "secondary",
			},
		},
	}
}

// NewDeleteConfirmDialog creates a confirmation dialog for delete operations.
func NewDeleteConfirmDialog(itemType, itemName string, payload any) *UIComponent {
	return NewConfirmDialogWithLabels(
		"确认删除",
		"确定要删除"+itemType+"「"+itemName+"」吗？此操作不可撤销。",
		"删除",
		"取消",
		payload,
		true,
	)
}
