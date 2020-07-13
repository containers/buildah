package util

import (
	"fmt"
	"testing"
)

func TestMergeEnv(t *testing.T) {
	tests := [][3][]string{
		{
			[]string{"A=B", "B=C", "C=D"},
			nil,
			[]string{"A=B", "B=C", "C=D"},
		},
		{
			nil,
			[]string{"A=B", "B=C", "C=D"},
			[]string{"A=B", "B=C", "C=D"},
		},
		{
			[]string{"A=B", "B=C", "C=D", "E=F"},
			[]string{"B=O", "F=G"},
			[]string{"A=B", "B=O", "C=D", "E=F", "F=G"},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			result := MergeEnv(test[0], test[1])
			if len(result) != len(test[2]) {
				t.Fatalf("expected %v, got %v", test[2], result)
			}
			for i := range result {
				if result[i] != test[2][i] {
					t.Fatalf("expected %v, got %v", test[2], result)
				}
			}
		})
	}
}
