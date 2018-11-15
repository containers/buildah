package idtools

import (
	"fmt"
	"strconv"
	"strings"
)

func nonDigitsToWhitespace(r rune) rune {
	if !strings.ContainsRune("0123456789", r) {
		return ' '
	}
	return r
}

func parseTriple(spec []string) (container, host, size uint32, err error) {
	cid, err := strconv.ParseUint(spec[0], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error parsing id map value %q: %v", spec[0], err)
	}
	hid, err := strconv.ParseUint(spec[1], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error parsing id map value %q: %v", spec[1], err)
	}
	sz, err := strconv.ParseUint(spec[2], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error parsing id map value %q: %v", spec[2], err)
	}
	return uint32(cid), uint32(hid), uint32(sz), nil
}

// ParseIDMap parses idmap triples from string.
func ParseIDMap(idMapSpec, mapSetting string) (idmap []IDMap, err error) {
	if len(idMapSpec) > 0 {
		idSpec := strings.Fields(strings.Map(nonDigitsToWhitespace, idMapSpec))
		if len(idSpec)%3 != 0 {
			return nil, fmt.Errorf("error initializing ID mappings: %s setting is malformed", mapSetting)
		}
		for i := range idSpec {
			if i%3 != 0 {
				continue
			}
			cid, hid, size, err := parseTriple(idSpec[i : i+3])
			if err != nil {
				return nil, fmt.Errorf("error initializing ID mappings: %s setting is malformed", mapSetting)
			}
			mapping := IDMap{
				ContainerID: int(cid),
				HostID:      int(hid),
				Size:        int(size),
			}
			idmap = append(idmap, mapping)
		}
	}
	return idmap, nil
}
