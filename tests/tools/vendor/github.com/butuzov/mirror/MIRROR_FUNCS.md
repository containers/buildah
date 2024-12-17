<tr>
<td><code>func (*bufio.Writer) Write([]byte) (int, error)</code></td>
<td><code>func (*bufio.Writer) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func (*bufio.Writer) WriteRune(rune) (int, error)</code></td>
<td><code>func (*bufio.Writer) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func (*bytes.Buffer) Write([]byte) (int, error)</code></td>
<td><code>func (*bytes.Buffer) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func (*bytes.Buffer) WriteRune(rune) (int, error)</code></td>
<td><code>func (*bytes.Buffer) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func bytes.Compare([]byte, []byte) int</code></td>
<td><code>func strings.Compare(string, string) int</code></td>
</tr>
<tr>
<td><code>func bytes.Contains([]byte, []byte) bool</code></td>
<td><code>func strings.Contains(string, string) bool</code></td>
</tr>
<tr>
<td><code>func bytes.ContainsAny([]byte, string) bool</code></td>
<td><code>func strings.ContainsAny(string, string) bool</code></td>
</tr>
<tr>
<td><code>func bytes.ContainsRune([]byte, byte) bool</code></td>
<td><code>func strings.ContainsRune(string, byte) bool</code></td>
</tr>
<tr>
<td><code>func bytes.Count([]byte, []byte) int</code></td>
<td><code>func strings.Count(string, string) int</code></td>
</tr>
<tr>
<td><code>func bytes.EqualFold([]byte, []byte) bool</code></td>
<td><code>func strings.EqualFold(string, string) bool</code></td>
</tr>
<tr>
<td><code>func bytes.HasPrefix([]byte, []byte) bool</code></td>
<td><code>func strings.HasPrefix(string, string) bool</code></td>
</tr>
<tr>
<td><code>func bytes.HasSuffix([]byte, []byte) bool</code></td>
<td><code>func strings.HasSuffix(string, string) bool</code></td>
</tr>
<tr>
<td><code>func bytes.Index([]byte, []byte) int</code></td>
<td><code>func strings.Index(string, string) int</code></td>
</tr>
<tr>
<td><code>func bytes.IndexAny([]byte, string) int</code></td>
<td><code>func strings.IndexAny(string, string) int</code></td>
</tr>
<tr>
<td><code>func bytes.IndexByte([]byte, byte) int</code></td>
<td><code>func strings.IndexByte(string, byte) int</code></td>
</tr>
<tr>
<td><code>func bytes.IndexFunc([]byte, func(rune) bool) int</code></td>
<td><code>func strings.IndexFunc(string, func(rune) bool) int</code></td>
</tr>
<tr>
<td><code>func bytes.IndexRune([]byte, rune) int</code></td>
<td><code>func strings.IndexRune(string, rune) int</code></td>
</tr>
<tr>
<td><code>func bytes.LastIndex([]byte, []byte) int</code></td>
<td><code>func strings.LastIndex(string, string) int</code></td>
</tr>
<tr>
<td><code>func bytes.LastIndexAny([]byte, string) int</code></td>
<td><code>func strings.LastIndexAny(string, string) int</code></td>
</tr>
<tr>
<td><code>func bytes.LastIndexByte([]byte, byte) int</code></td>
<td><code>func strings.LastIndexByte(string, byte) int</code></td>
</tr>
<tr>
<td><code>func bytes.LastIndexFunc([]byte, func(rune) bool) int</code></td>
<td><code>func strings.LastIndexFunc(string, func(rune) bool) int</code></td>
</tr>
<tr>
<td><code>func bytes.NewBuffer([]byte) *bytes.Buffer</code></td>
<td><code>func bytes.NewBufferString(string) *bytes.Buffer</code></td>
</tr>
<tr>
<td><code>func (*httptest.ResponseRecorder) Write([]byte) (int, error)</code></td>
<td><code>func (*httptest.ResponseRecorder) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func (*maphash.Hash) Write([]byte) (int, error)</code></td>
<td><code>func (*maphash.Hash) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func (*os.File) Write([]byte) (int, error)</code></td>
<td><code>func (*os.File) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func regexp.Match(string, []byte) (bool, error)</code></td>
<td><code>func regexp.MatchString(string, string) (bool, error)</code></td>
</tr>
<tr>
<td><code>func (*regexp.Regexp) FindAllIndex([]byte, int) [][]int</code></td>
<td><code>func (*regexp.Regexp) FindAllStringIndex(string, int) [][]int</code></td>
</tr>
<tr>
<td><code>func (*regexp.Regexp) FindAllSubmatchIndex([]byte, int) [][]int</code></td>
<td><code>func (*regexp.Regexp) FindAllStringSubmatchIndex(string, int) [][]int</code></td>
</tr>
<tr>
<td><code>func (*regexp.Regexp) FindIndex([]byte) []int</code></td>
<td><code>func (*regexp.Regexp) FindStringIndex(string) []int</code></td>
</tr>
<tr>
<td><code>func (*regexp.Regexp) FindSubmatchIndex([]byte) []int</code></td>
<td><code>func (*regexp.Regexp) FindStringSubmatchIndex(string) []int</code></td>
</tr>
<tr>
<td><code>func (*regexp.Regexp) Match([]byte) bool</code></td>
<td><code>func (*regexp.Regexp) MatchString(string) bool</code></td>
</tr>
<tr>
<td><code>func (*strings.Builder) Write([]byte) (int, error)</code></td>
<td><code>func (*strings.Builder) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func (*strings.Builder) WriteRune(rune) (int, error)</code></td>
<td><code>func (*strings.Builder) WriteString(string) (int, error)</code></td>
</tr>
<tr>
<td><code>func strings.Compare(string) int</code></td>
<td><code>func bytes.Compare([]byte) int</code></td>
</tr>
<tr>
<td><code>func strings.Contains(string) bool</code></td>
<td><code>func bytes.Contains([]byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.ContainsAny(string) bool</code></td>
<td><code>func bytes.ContainsAny([]byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.ContainsRune(string) bool</code></td>
<td><code>func bytes.ContainsRune([]byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.EqualFold(string) bool</code></td>
<td><code>func bytes.EqualFold([]byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.HasPrefix(string) bool</code></td>
<td><code>func bytes.HasPrefix([]byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.HasSuffix(string) bool</code></td>
<td><code>func bytes.HasSuffix([]byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.Index(string) int</code></td>
<td><code>func bytes.Index([]byte) int</code></td>
</tr>
<tr>
<td><code>func strings.IndexFunc(string, func(r rune) bool) int</code></td>
<td><code>func bytes.IndexFunc([]byte, func(r rune) bool) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndex(string) int</code></td>
<td><code>func bytes.LastIndex([]byte) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndexAny(string) int</code></td>
<td><code>func bytes.LastIndexAny([]byte) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndexFunc(string, func(r rune) bool) int</code></td>
<td><code>func bytes.LastIndexFunc([]byte, func(r rune) bool) int</code></td>
</tr>
<tr>
<td><code>func utf8.DecodeLastRune([]byte) (rune, int)</code></td>
<td><code>func utf8.DecodeLastRuneInString(string) (rune, int)</code></td>
</tr>
<tr>
<td><code>func utf8.DecodeRune([]byte) (rune, int)</code></td>
<td><code>func utf8.DecodeRuneInString(string) (rune, int)</code></td>
</tr>
<tr>
<td><code>func utf8.FullRune([]byte) bool</code></td>
<td><code>func utf8.FullRuneInString(string) bool</code></td>
</tr>
<tr>
<td><code>func utf8.RuneCount([]byte) int</code></td>
<td><code>func utf8.RuneCountInString(string) int</code></td>
</tr>
<tr>
<td><code>func utf8.Valid([]byte) bool</code></td>
<td><code>func utf8.ValidString(string) bool</code></td>
</tr>

