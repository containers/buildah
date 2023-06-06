<table><tr>
<td><code>func (b *bufio.Writer) WriteString(s string) (int, error)</code></td>
<td>
  <code>func (b *bufio.Writer) Write(p []byte) (int, error)</code>
  <code>func (b *bufio.Writer) WriteRune(r rune) (int, error)</code>
</td>
</tr>
<tr>
<td><code>func (b *bytes.Buffer) WriteString(s string) (int, error)</code></td>
<td>
  <code>func (b *bytes.Buffer) Write(p []byte) (int, error)</code>
  <code>func (b *bytes.Buffer) WriteRune(r rune) (int, error)</code>
 </td>
</tr>
<tr>
<td><code>func strings.Compare(a, b string) int</code></td>
<td><code>func bytes.Compare(a, b []byte) int</code></td>
</tr>
<tr>
<td><code>func strings.Contains(s, substr string) bool</code></td>
<td><code>func bytes.Contains(b, subslice []byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.ContainsAny(s, chars string) bool</code></td>
<td><code>func bytes.ContainsAny(b []byte, chars string) bool</code></td>
</tr>
<tr>
<td><code>func strings.ContainsRune(s string, r rune) bool</code></td>
<td><code>func bytes.ContainsRune(b []byte, r rune) bool</code></td>
</tr>
<tr>
<td><code>func strings.Count(s, substr string) int</code></td>
<td><code>func bytes.Count(s, sep []byte) int</code></td>
</tr>
<tr>
<td><code>func strings.EqualFold(s, t string) bool</code></td>
<td><code>func bytes.EqualFold(s, t []byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.HasPrefix(s, prefix string) bool</code></td>
<td><code>func bytes.HasPrefix(s, prefix []byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.HasSuffix(s, suffix string) bool</code></td>
<td><code>func bytes.HasSuffix(s, suffix []byte) bool</code></td>
</tr>
<tr>
<td><code>func strings.Index(s, substr string) int</code></td>
<td><code>func bytes.Index(s, sep []byte) int</code></td>
</tr>
<tr>
<td><code>func strings.IndexAny(s, chars string) int</code></td>
<td><code>func bytes.IndexAny(s []byte, chars string) int</code></td>
</tr>
<tr>
<td><code>func strings.IndexByte(s string, c byte) int</code></td>
<td><code>func bytes.IndexByte(b []byte, c byte) int</code></td>
</tr>
<tr>
<td><code>func strings.IndexFunc(s string, f func(rune) bool) int</code></td>
<td><code>func bytes.IndexFunc(s []byte, f func(r rune) bool) int</code></td>
</tr>
<tr>
<td><code>func strings.IndexRune(s string, r rune) int</code></td>
<td><code>func bytes.IndexRune(s []byte, r rune) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndex(s, sep string) int</code></td>
<td><code>func bytes.LastIndex(s, sep []byte) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndexAny(s, chars string) int</code></td>
<td><code>func bytes.LastIndexAny(s []byte, chars string) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndexByte(s string, c byte) int</code></td>
<td><code>func bytes.LastIndexByte(s []byte, c byte) int</code></td>
</tr>
<tr>
<td><code>func strings.LastIndexFunc(s string, f func(rune) bool) int</code></td>
<td><code>func bytes.LastIndexFunc(s []byte, f func(r rune) bool) int</code></td>
</tr>
<tr>
<td><code>func bytes.NewBufferString(s string) *bytes.Buffer</code></td>
<td><code>func bytes.NewBuffer(buf []byte *bytes.Buffer</code></td>
</tr>
<tr>
<td><code>func (h *hash/maphash.Hash) WriteString(s string) (int, error)</code></td>
<td><code>func (h *hash/maphash.Hash) Write(b []byte) (int, error)</code></td>
</tr>
<tr>
<td><code>func (rw *net/http/httptest.ResponseRecorder) WriteString(str string) (int, error)</code></td>
<td><code>func (rw *net/http/httptest.ResponseRecorder) Write(buf []byte) (int, error)</code></td>
</tr>
<tr>
<td><code>func (f *os.File) WriteString(s string) (n int, err error)</code></td>
<td><code>func (f *os.File) Write(b []byte) (n int, err error)</code></td>
</tr>
<tr>
<td><code>func regexp.MatchString(pattern string, s string) (bool, error)</code></td>
<td><code>func regexp.Match(pattern string, b []byte) (bool, error)</code></td>
</tr>
<tr>
<td><code>func (re *regexp.Regexp) FindAllStringIndex(s string, n int) [][]int</code></td>
<td><code>func (re *regexp.Regexp) FindAllIndex(b []byte, n int) [][]int</code></td>
</tr>
<tr>
<td><code>func (re *regexp.Regexp) FindAllStringSubmatch(s string, n int) [][]string</code></td>
<td><code>func (re *regexp.Regexp) FindAllSubmatch(b []byte, n int) [][][]byte</code></td>
</tr>
<tr>
<td><code>func (re *regexp.Regexp) FindStringIndex(s string) (loc []int)</code></td>
<td><code>func (re *regexp.Regexp) FindIndex(b []byte) (loc []int)</code></td>
</tr>
<tr>
<td><code>func (re *regexp.Regexp) FindStringSubmatchIndex(s string) []int</code></td>
<td><code>func (re *regexp.Regexp) FindSubmatchIndex(b []byte) []int</code></td>
</tr>
<tr>
<td><code>func (re *regexp.Regexp) MatchString(s string) bool</code></td>
<td><code>func (re *regexp.Regexp) Match(b []byte) bool</code></td>
</tr>
<tr>
<td><code>func (b *strings.Builder) WriteString(s string) error</code></td>
<td>
  <code>func (b *strings.Builder) Write(p []byte) (int, error)</code>
  <code>func (b *strings.Builder) WriteRune(r rune) (int, error)</code>
 </td>
</tr>
<tr>
<td><code>func utf8.ValidString(s string) bool</code></td>
<td><code>func utf8.Valid(p []byte) bool</code></td>
</tr>
<tr>
<td><code>func utf8.FullRuneInString(s string) bool</code></td>
<td><code>func utf8.FullRune(p []byte) bool</code></td>
</tr>
<tr>
<td><code>func utf8.RuneCountInString(s string) (n int)</code></td>
<td><code>func utf8.RuneCount(p []byte) int</code></td>
</tr>
<tr>
<td><code>func utf8.DecodeLastRuneInString(s string) (rune, int)</code></td>
<td><code>func utf8.DecodeLastRune(p []byte) (rune, int)</code></td>
</tr>
<tr>
<td><code>func utf8.DecodeRuneInString(s string) (une, int)</code></td>
<td><code>func utf8.DecodeRune(p []byte) (rune, int)</code></td>
</tr>
</table>
