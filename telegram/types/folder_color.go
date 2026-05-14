package types

// FolderColor enumerates the available colors for a chat folder. Chat folders
// can be assigned a color to visually distinguish them in the folder tab bar.
type FolderColor string

const (
	// FolderColorBlue applies a blue color to the chat folder tab.
	FolderColorBlue FolderColor = "blue"
	// FolderColorCyan applies a cyan color to the chat folder tab.
	FolderColorCyan FolderColor = "cyan"
	// FolderColorGreen applies a green color to the chat folder tab.
	FolderColorGreen FolderColor = "green"
	// FolderColorOrange applies an orange color to the chat folder tab.
	FolderColorOrange FolderColor = "orange"
	// FolderColorPink applies a pink color to the chat folder tab.
	FolderColorPink FolderColor = "pink"
	// FolderColorRed applies a red color to the chat folder tab.
	FolderColorRed FolderColor = "red"
	// FolderColorViolet applies a violet color to the chat folder tab.
	FolderColorViolet FolderColor = "violet"
)

// String returns the string representation of the FolderColor.
func (f FolderColor) String() string { return string(f) }
