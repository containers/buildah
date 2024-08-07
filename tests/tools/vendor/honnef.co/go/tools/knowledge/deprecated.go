package knowledge

const (
	// DeprecatedNeverUse indicates that an API should never be used, regardless of Go version.
	DeprecatedNeverUse = -1
	// DeprecatedUseNoLonger indicates that an API has no use anymore.
	DeprecatedUseNoLonger = -2
)

// Deprecation describes when a Go API has been deprecated.
type Deprecation struct {
	// The minor Go version since which this API has been deprecated.
	DeprecatedSince int
	// The minor Go version since which an alternative API has been available.
	// May also be one of DeprecatedNeverUse or DeprecatedUseNoLonger.
	AlternativeAvailableSince int
}

// go/importer.ForCompiler contains "Deprecated:", but it refers to a single argument, not the whole function.
// Luckily, the notice starts in the middle of a paragraph, and as such isn't detected by us.

// TODO(dh): StdlibDeprecations doesn't contain entries for internal packages and unexported API. That's fine for normal
// users, but makes the Deprecated check less useful for people working on Go itself.

// StdlibDeprecations contains a mapping of Go API (such as variables, methods, or fields, among others)
// to information about when it has been deprecated.
var StdlibDeprecations = map[string]Deprecation{
	// FIXME(dh): AllowBinary isn't being detected as deprecated
	// because the comment has a newline right after "Deprecated:"
	"go/build.AllowBinary":                      {7, 7},
	"(archive/zip.FileHeader).CompressedSize":   {1, 1},
	"(archive/zip.FileHeader).UncompressedSize": {1, 1},
	"(archive/zip.FileHeader).ModifiedTime":     {10, 10},
	"(archive/zip.FileHeader).ModifiedDate":     {10, 10},
	"(*archive/zip.FileHeader).ModTime":         {10, 10},
	"(*archive/zip.FileHeader).SetModTime":      {10, 10},
	"(go/doc.Package).Bugs":                     {1, 1},
	"os.SEEK_SET":                               {7, 7},
	"os.SEEK_CUR":                               {7, 7},
	"os.SEEK_END":                               {7, 7},
	"(net.Dialer).Cancel":                       {7, 7},
	"runtime.CPUProfile":                        {9, 0},
	"compress/flate.ReadError":                  {6, DeprecatedUseNoLonger},
	"compress/flate.WriteError":                 {6, DeprecatedUseNoLonger},
	"path/filepath.HasPrefix":                   {0, DeprecatedNeverUse},
	"(net/http.Transport).Dial":                 {7, 7},
	"(net/http.Transport).DialTLS":              {14, 14},
	"(*net/http.Transport).CancelRequest":       {6, 5},
	"net/http.ErrWriteAfterFlush":               {7, DeprecatedUseNoLonger},
	"net/http.ErrHeaderTooLong":                 {8, DeprecatedUseNoLonger},
	"net/http.ErrShortBody":                     {8, DeprecatedUseNoLonger},
	"net/http.ErrMissingContentLength":          {8, DeprecatedUseNoLonger},
	"net/http/httputil.ErrPersistEOF":           {0, DeprecatedUseNoLonger},
	"net/http/httputil.ErrClosed":               {0, DeprecatedUseNoLonger},
	"net/http/httputil.ErrPipeline":             {0, DeprecatedUseNoLonger},
	"net/http/httputil.ServerConn":              {0, 0},
	"net/http/httputil.NewServerConn":           {0, 0},
	"net/http/httputil.ClientConn":              {0, 0},
	"net/http/httputil.NewClientConn":           {0, 0},
	"net/http/httputil.NewProxyClientConn":      {0, 0},
	"(net/http.Request).Cancel":                 {7, 7},
	"(text/template/parse.PipeNode).Line":       {1, DeprecatedUseNoLonger},
	"(text/template/parse.ActionNode).Line":     {1, DeprecatedUseNoLonger},
	"(text/template/parse.BranchNode).Line":     {1, DeprecatedUseNoLonger},
	"(text/template/parse.TemplateNode).Line":   {1, DeprecatedUseNoLonger},
	"database/sql/driver.ColumnConverter":       {9, 9},
	"database/sql/driver.Execer":                {8, 8},
	"database/sql/driver.Queryer":               {8, 8},
	"(database/sql/driver.Conn).Begin":          {8, 8},
	"(database/sql/driver.Stmt).Exec":           {8, 8},
	"(database/sql/driver.Stmt).Query":          {8, 8},
	"syscall.StringByteSlice":                   {1, 1},
	"syscall.StringBytePtr":                     {1, 1},
	"syscall.StringSlicePtr":                    {1, 1},
	"syscall.StringToUTF16":                     {1, 1},
	"syscall.StringToUTF16Ptr":                  {1, 1},
	"(*regexp.Regexp).Copy":                     {12, DeprecatedUseNoLonger},
	"(archive/tar.Header).Xattrs":               {10, 10},
	"archive/tar.TypeRegA":                      {11, 1},
	"go/types.NewInterface":                     {11, 11},
	"(*go/types.Interface).Embedded":            {11, 11},
	"go/importer.For":                           {12, 12},
	"encoding/json.InvalidUTF8Error":            {2, DeprecatedUseNoLonger},
	"encoding/json.UnmarshalFieldError":         {2, DeprecatedUseNoLonger},
	"encoding/csv.ErrTrailingComma":             {2, DeprecatedUseNoLonger},
	"(encoding/csv.Reader).TrailingComma":       {2, DeprecatedUseNoLonger},
	"(net.Dialer).DualStack":                    {12, 12},
	"net/http.ErrUnexpectedTrailer":             {12, DeprecatedUseNoLonger},
	"net/http.CloseNotifier":                    {11, 7},
	// This is hairy. The notice says "Not all errors in the http package related to protocol errors are of type ProtocolError", but doesn't that imply that some errors do?
	"net/http.ProtocolError":                       {8, DeprecatedUseNoLonger},
	"(crypto/x509.CertificateRequest).Attributes":  {5, 3},
	"(*crypto/x509.Certificate).CheckCRLSignature": {19, 19},
	"crypto/x509.ParseCRL":                         {19, 19},
	"crypto/x509.ParseDERCRL":                      {19, 19},
	"(*crypto/x509.Certificate).CreateCRL":         {19, 19},
	"crypto/x509/pkix.TBSCertificateList":          {19, 19},
	"crypto/x509/pkix.RevokedCertificate":          {19, 19},
	"go/doc.ToHTML":                                {20, 20},
	"go/doc.ToText":                                {20, 20},
	"go/doc.Synopsis":                              {20, 20},
	"math/rand.Seed":                               {20, 0},
	"math/rand.Read":                               {20, DeprecatedNeverUse},

	// These functions have no direct alternative, but they are insecure and should no longer be used.
	"crypto/x509.IsEncryptedPEMBlock": {16, DeprecatedNeverUse},
	"crypto/x509.DecryptPEMBlock":     {16, DeprecatedNeverUse},
	"crypto/x509.EncryptPEMBlock":     {16, DeprecatedNeverUse},
	"crypto/dsa":                      {16, DeprecatedNeverUse},

	// This function has no alternative, but also no purpose.
	"(*crypto/rc4.Cipher).Reset":                     {12, DeprecatedNeverUse},
	"(net/http/httptest.ResponseRecorder).HeaderMap": {11, 7},
	"image.ZP":                                    {13, 0},
	"image.ZR":                                    {13, 0},
	"(*debug/gosym.LineTable).LineToPC":           {2, 2},
	"(*debug/gosym.LineTable).PCToLine":           {2, 2},
	"crypto/tls.VersionSSL30":                     {13, DeprecatedNeverUse},
	"(crypto/tls.Config).NameToCertificate":       {14, DeprecatedUseNoLonger},
	"(*crypto/tls.Config).BuildNameToCertificate": {14, DeprecatedUseNoLonger},
	"(crypto/tls.Config).SessionTicketKey":        {16, 5},
	// No alternative, no use
	"(crypto/tls.ConnectionState).NegotiatedProtocolIsMutual": {16, DeprecatedNeverUse},
	// No alternative, but insecure
	"(crypto/tls.ConnectionState).TLSUnique": {16, DeprecatedNeverUse},
	"image/jpeg.Reader":                      {4, DeprecatedNeverUse},

	// All of these have been deprecated in favour of external libraries
	"syscall.AttachLsf":                     {7, 0},
	"syscall.DetachLsf":                     {7, 0},
	"syscall.LsfSocket":                     {7, 0},
	"syscall.SetLsfPromisc":                 {7, 0},
	"syscall.LsfJump":                       {7, 0},
	"syscall.LsfStmt":                       {7, 0},
	"syscall.BpfStmt":                       {7, 0},
	"syscall.BpfJump":                       {7, 0},
	"syscall.BpfBuflen":                     {7, 0},
	"syscall.SetBpfBuflen":                  {7, 0},
	"syscall.BpfDatalink":                   {7, 0},
	"syscall.SetBpfDatalink":                {7, 0},
	"syscall.SetBpfPromisc":                 {7, 0},
	"syscall.FlushBpf":                      {7, 0},
	"syscall.BpfInterface":                  {7, 0},
	"syscall.SetBpfInterface":               {7, 0},
	"syscall.BpfTimeout":                    {7, 0},
	"syscall.SetBpfTimeout":                 {7, 0},
	"syscall.BpfStats":                      {7, 0},
	"syscall.SetBpfImmediate":               {7, 0},
	"syscall.SetBpf":                        {7, 0},
	"syscall.CheckBpfVersion":               {7, 0},
	"syscall.BpfHeadercmpl":                 {7, 0},
	"syscall.SetBpfHeadercmpl":              {7, 0},
	"syscall.RouteRIB":                      {8, 0},
	"syscall.RoutingMessage":                {8, 0},
	"syscall.RouteMessage":                  {8, 0},
	"syscall.InterfaceMessage":              {8, 0},
	"syscall.InterfaceAddrMessage":          {8, 0},
	"syscall.ParseRoutingMessage":           {8, 0},
	"syscall.ParseRoutingSockaddr":          {8, 0},
	"syscall.InterfaceAnnounceMessage":      {7, 0},
	"syscall.InterfaceMulticastAddrMessage": {7, 0},
	"syscall.FormatMessage":                 {5, 0},
	"syscall.PostQueuedCompletionStatus":    {17, 0},
	"syscall.GetQueuedCompletionStatus":     {17, 0},
	"syscall.CreateIoCompletionPort":        {17, 0},

	// We choose to only track the package itself, even though all functions are derecated individually, too. Anyone
	// using ioutil directly will have to import it, and this keeps the noise down.
	"io/ioutil": {19, 19},

	"bytes.Title":   {18, 0},
	"strings.Title": {18, 0},
	"(crypto/tls.Config).PreferServerCipherSuites": {18, DeprecatedUseNoLonger},
	// It's not clear if Subjects was okay to use in the past, so we err on the less noisy side of assuming that it was.
	"(*crypto/x509.CertPool).Subjects": {18, DeprecatedUseNoLonger},
	"go/types.NewSignature":            {18, 18},
	"(net.Error).Temporary":            {18, DeprecatedNeverUse},
	// InterfaceData is another tricky case. It was deprecated in Go 1.18, but has been useless since Go 1.4, and an
	// "alternative" (using your own unsafe hacks) has existed forever. We don't want to get into hairsplitting with
	// users who somehow successfully used this between 1.4 and 1.18, so we'll just tag it as deprecated since 1.18.
	"(reflect.Value).InterfaceData": {18, 18},

	// The following objects are only deprecated on Windows.
	"syscall.Syscall":   {18, 18},
	"syscall.Syscall12": {18, 18},
	"syscall.Syscall15": {18, 18},
	"syscall.Syscall18": {18, 18},
	"syscall.Syscall6":  {18, 18},
	"syscall.Syscall9":  {18, 18},

	"reflect.SliceHeader":                              {21, 17},
	"reflect.StringHeader":                             {21, 20},
	"crypto/elliptic.GenerateKey":                      {21, 21},
	"crypto/elliptic.Marshal":                          {21, 21},
	"crypto/elliptic.Unmarshal":                        {21, 21},
	"(*crypto/elliptic.CurveParams).Add":               {21, 21},
	"(*crypto/elliptic.CurveParams).Double":            {21, 21},
	"(*crypto/elliptic.CurveParams).IsOnCurve":         {21, 21},
	"(*crypto/elliptic.CurveParams).ScalarBaseMult":    {21, 21},
	"(*crypto/elliptic.CurveParams).ScalarMult":        {21, 21},
	"crypto/rsa.GenerateMultiPrimeKey":                 {21, DeprecatedNeverUse},
	"(crypto/rsa.PrecomputedValues).CRTValues":         {21, DeprecatedNeverUse},
	"(crypto/x509.RevocationList).RevokedCertificates": {21, 21},
}

// Last imported from Go at c19c4c566c63818dfd059b352e52c4710eecf14d with the following numbers of deprecations:
//
// archive/tar/common.go:2
// archive/zip/struct.go:6
// bytes/bytes.go:1
// compress/flate/inflate.go:2
// crypto/dsa/dsa.go:1
// crypto/elliptic/elliptic.go:8
// crypto/elliptic/params.go:5
// crypto/rc4/rc4.go:1
// crypto/rsa/rsa.go:2
// crypto/tls/common.go:6
// crypto/x509/cert_pool.go:1
// crypto/x509/pem_decrypt.go:3
// crypto/x509/pkix/pkix.go:2
// crypto/x509/x509.go:6
// database/sql/driver/driver.go:6
// debug/gosym/pclntab.go:2
// encoding/csv/reader.go:2
// encoding/json/decode.go:1
// encoding/json/encode.go:1
// go/build/build.go:1
// go/doc/comment.go:2
// go/doc/doc.go:1
// go/doc/synopsis.go:1
// go/importer/importer.go:2
// go/types/interface.go:2
// go/types/signature.go:1
// image/geom.go:2
// image/jpeg/reader.go:1
// internal/types/errors/codes.go:1
// io/ioutil/ioutil.go:7
// io/ioutil/tempfile.go:2
// math/rand/rand.go:2
// net/dial.go:2
// net/http/h2_bundle.go:1
// net/http/httptest/recorder.go:1
// net/http/httputil/persist.go:8
// net/http/request.go:6
// net/http/server.go:2
// net/http/socks_bundle.go:1
// net/http/transport.go:3
// net/net.go:1
// os/file.go:1
// path/filepath/path_plan9.go:1
// path/filepath/path_unix.go:1
// path/filepath/path_windows.go:1
// reflect/value.go:3
// regexp/regexp.go:1
// runtime/cpuprof.go:1
// strings/strings.go:1
// syscall/bpf_bsd.go:18
// syscall/bpf_darwin.go:18
// syscall/dll_windows.go:6
// syscall/exec_plan9.go:1
// syscall/exec_unix.go:1
// syscall/lsf_linux.go:6
// syscall/route_bsd.go:7
// syscall/route_darwin.go:1
// syscall/route_dragonfly.go:2
// syscall/route_freebsd.go:2
// syscall/route_netbsd.go:1
// syscall/route_openbsd.go:1
// syscall/syscall.go:3
// syscall/syscall_windows.go:6
// text/template/parse/node.go:5
// vendor/golang.org/x/text/transform/transform.go:1
