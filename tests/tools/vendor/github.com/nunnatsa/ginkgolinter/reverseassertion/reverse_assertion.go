package reverseassertion

var reverseLogicAssertions = map[string]string{
	"To":        "ToNot",
	"ToNot":     "To",
	"NotTo":     "To",
	"Should":    "ShouldNot",
	"ShouldNot": "Should",
}

// ChangeAssertionLogic get gomega assertion function name, and returns the reverse logic function name
func ChangeAssertionLogic(funcName string) string {
	if revFunc, ok := reverseLogicAssertions[funcName]; ok {
		return revFunc
	}
	return funcName
}
