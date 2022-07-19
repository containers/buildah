package section

import (
	"errors"
	"fmt"
	"strings"
)

func Parse(data []string) (SectionList, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var list SectionList
	var errString string
	for _, d := range data {
		s := strings.ToLower(d)
		if len(s) == 0 {
			return nil, nil
		}

		if s == "default" {
			list = append(list, Default{})
		} else if s == "standard" {
			list = append(list, Standard{})
		} else if s == "newline" {
			list = append(list, NewLine{})
		} else if strings.HasPrefix(s, "prefix(") && len(s) > 8 {
			list = append(list, Custom{s[7 : len(s)-1]})
		} else if strings.HasPrefix(s, "commentline(") && len(s) > 13 {
			list = append(list, Custom{s[12 : len(s)-1]})
		} else {
			errString += fmt.Sprintf(" %s", s)
		}
	}
	if errString != "" {
		return nil, errors.New(fmt.Sprintf("invalid params:%s", errString))
	}
	return list, nil
}
