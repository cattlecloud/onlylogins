package sitecontent

import (
	"errors"
	"regexp"

	"cattlecloud.net/onlylogins/database/records"
)

// A Folder is the categorization unit for each user to store
// bits of data into.
type Folder struct {
	ID    records.FoldersAuto
	Label string
	Icon  Icon
}

// Icon is an enum associated with an SVG image.
type Icon int

var FolderIcons = map[Icon]string{
	0:  "attention.svg",
	1:  "heart.svg",
	2:  "house.svg",
	3:  "save.svg",
	4:  "search.svg",
	5:  "share.svg",
	6:  "square.svg",
	7:  "stack.svg",
	8:  "stock.svg",
	9:  "tag.svg",
	10: "thumbup.svg",
	11: "tower.svg",
}

// labelRe restricts folder labels to alphanumeric characters so they may be
// used directly as URL path elements without requiring percent-encoding.
var labelRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// ValidateLabel returns an error if s is not a usable folder label. A valid
// label consists of one or more alphanumeric characters and is therefore
// safe to embed in a URL path segment.
func ValidateLabel(s string) error {
	if !labelRe.MatchString(s) {
		return errors.New("label must contain only alphanumeric characters")
	}
	return nil
}
