// Copyright 2016 The Linux Foundation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// _escFS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func _escFS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// _escDir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func _escDir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// _escFSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func _escFSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// _escFSMustByte is the same as _escFSByte, but panics if name is not present.
func _escFSMustByte(useLocal bool, name string) []byte {
	b, err := _escFSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// _escFSString is the string version of _escFSByte.
func _escFSString(useLocal bool, name string) (string, error) {
	b, err := _escFSByte(useLocal, name)
	return string(b), err
}

// _escFSMustString is the string version of _escFSMustByte.
func _escFSMustString(useLocal bool, name string) string {
	return string(_escFSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/config-schema.json": {
		local:   "config-schema.json",
		size:    774,
		modtime: 1479372487,
		compressed: `
H4sIAAAJbogA/5SRsW4DIQyG93sKRDL2QodOt+YBOnSsOlAwOUc6TI0zRFXevQJyTU6KquvCYP/f/2P7
u1NKe8iOMQlS1IPSrwninqJYjMBqTzHgQb0lcBjQ2ap6Ktg2uxEmW5BRJA3GHDPFvlV3xAfj2Qbpn19M
q20ah35G8mAMJYhuTssVa2qDkz2AcTW/kXJOUFj6PIKTVktMCVgQsh5UGUcp7RisgP8t3KFZGGOzq/VA
PFkpHW8FesEJdO1dmkTbk4zEf1gt1exGFHByYljLUF6rvO7iTr1lCPXzEHLf2rtyhY3xEDBiuVaed7jw
YiIJ+f9eV27hNWIW4vOjMSyzPd/WjQLTfejq2Dniyl26+a1/0AxfJ+R68vcHl7htejl9p9RHd+l+AgAA
//9eW+CxBgMAAA==
`,
	},

	"/content-descriptor.json": {
		local:   "content-descriptor.json",
		size:    836,
		modtime: 1479372487,
		compressed: `
H4sIAAAJbogA/5SSv+/TMBDF9/wVJ7cjrRkqhqywM8CGGFz7nFzV2OZ8VRVQ/3fkOGlTkL4/lih6ep+7
92z/aQCUw2yZklAMqgX1NWH4HIMYCshQ/jAIfJlNkeFbQkuerJmID2XENtseB1PwXiS1Wp9yDLuq7iN3
2rHxsvt40FXbVI7cguRW65gw2GVznrDq1u6+vXIyJixkPJ7QStUSx4QshFm1UIoBqAEdme/VXKX/+0qP
MPnKUIgeisDokTFYdLDaMeFbRl84hz7vaDAd7kvZjUNPgcrUrB97J+pWYZXp92tJigUowHEUzO9MU3Po
dRAK8unwHMJRh1nWMd7SaKaeJl34nF+oY+BMWUqHYgTPcYBrT7YH6SnPTWAwIxwRXLyGczQO3aPbcsuG
2YwPmQSH9d6VMwtT6O5WAOUjD6a0VRcmNeu3ZvlOVRTjrwsxlsf449+Hs7665xNsAH42t+ZvAAAA//9L
DLQ9RAMAAA==
`,
	},

	"/defs-config.json": {
		local:   "defs-config.json",
		size:    2483,
		modtime: 1480556605,
		compressed: `
H4sIAAAJbogA/+RWzY7TMBC+5ykiw7GwF8SB6y43UJEi4IBQ5SaT7Syxx4wnQIT23XGypRsnrel2tycO
VRvH389845/+zvJcVeBLRidIVr3J1RXUaLF/8rnTLFi2jeZcKF86sJdkRaMFzsOvGq/zwkGJNZZ6wC/u
CHcMgbDXCIPlMH33HEakc9AL0voGShmgw7hjchB0wY9mh/GPHjgaGXF4YbTXavfqdnGPew+GuJsinzPU
PTKY9S9vPNlnFyPfF2jl9asUX/FTu6fjvHRtsdE8KflRlG9/OfJQfSCWh7Ia7Yoh0OVda/bz2x+HuqGZ
dacW41coYKY2Ev0LUgdEhTtHoeypNllY9jV9iQRiuaTFhM1/WI3tTiwnXdi2aWKmMc/X/UvFVP9t7Z+o
ac18kxxZ/6mr/txVfSb+FmSv8KTj7Z1eQ3PuSIo9jX/ySLLx95ZdMZHU/jH3RoV1vcJqFtH5T6vt/FRL
I1mwrZn1TDW6A/YqndkuBYbvLTJUEc99BlN32Zjxb+Yb9BJfmQ8OvWTQArOTKlV9TWy0DKsxQF8IGti/
4nUrGzppr2xdrdazvwNHockYmN88x0DBOOlWQycPwddEDWirDu2HrP/cZn8CAAD//0zDaxqzCQAA
`,
	},

	"/defs-image.json": {
		local:   "defs-image.json",
		size:    2736,
		modtime: 1480556516,
		compressed: `
H4sIAAAJbogA/7yWz27bOBDG736KgRIgB1tWDosFYgQBFpvLnnLInmq4xYQaWZNKpDqkkzqB372gLMv6
m8BN0JMhcvjx+w1nSL9OAIKYrBIuHBsdLCC4pYQ1+y8LBYpjtclQwBm4K0j/a7RD1iTwX45rgvuCFCes
sFw+2+vVAsEC/BYAQU4x4//bguohgIBjv2HqXGEXUWQK0uogb+dG1pFVKeUYsd8qOkrMDgJuLxhYJ6zX
x/ECnSMpcb4uMXxZTaPlZXiF4cs/4Zf5dDU9D8rQ3X5FEPOarGta6yTFpQRKtoUza8EiZQUqJfXdbnLY
rwWTgA8yD4+k3AxYl5+VEbi4TtGmN4vrlH5iTIpzzKqVNxen81yGV9+m83A1XSwxTC7Dqx5RjpoTsu62
4jDygcT3tXqO99zHcaEfGxby+yyrsVYVzI6Dll9a39VhNEaKDF1iJA+qodUxMWIKEsdkG3gj5TZ2rmWs
5zicoVBCQlpRDB2sUuJcKPFrzxqF3ijOOnTXZXzfig/zpfOwdWRPtBNTYueP1uizqGmMtfv7r2FTvaof
x6tCB2U2ktl32BAy3veID4ZETA7PKasUXMq2woIct/BAEJtnnRmMKW6DHkoNRXDbnmJHedfDeDtVs76i
0OMHG+GgMbcbxKyLsIN6Wi/VKoNoQ+c72EnlDIpK2ZFyG6EunLFNolVLcaRp+prd2d+mjd6w2j+n1vSu
D/Z5toz9qJn5E4ndV/nnmapFP2ouIfQZ/9yUHVVH7fVbFMbb9F2wZkcOgT6hMOruRXZyvt5K1h8CG76E
Js3fwwOPWhuH7X9ZJ59qU6R+U42mu6R12wy+DyPvTY7FfYl63wFupPt16O7TmywLetCryeFrN9lNfgUA
AP//R591MrAKAAA=
`,
	},

	"/defs.json": {
		local:   "defs.json",
		size:    3193,
		modtime: 1470056192,
		compressed: `
H4sIAAAJbogA/7SWQZPSMBTH73yKTPSIbpumLXBzRNc97MDMjifHQy0PiEIS09Rx3eG7OymlNG0oFPGw
C03yfv/3/nmheRkghBeQpYpJzQTHE4SnsGScmacM5RkskF4rka/WItdIrwHNJPD3guuEcVDoSULKlixN
ivDhnlcB8AQZCYQw43pUPSGE9bMEo8a4hhWoIrKY2DLOtvkWT9Abn4yOw8nvctgncTG4G1ZkP+qLDkgc
ueBmvIUPSF888WlMR0FEXRrVZEsoon2FxoQEQUy8IBqFNI4jz/Mcio5VlnR+xe64dEgYtri998YFjsIw
aKN774sLTcmYjqOYjNv83tvh4vsjSqOYUi8OYm8chsRlvh/NheGquqDgMFviCfpSDqBqqph+rcDM4ld3
tQN3V1peLdwN3dGHani+2dRWl9++Ory4ZYYRvW2GmVaMr/pneNDYx982p20inwrs/r+rlcS375DqYyfJ
RGtQfK6EBKUZZLUghPDbF3+4s4a6ShjUP1tJzfbS/zmpknUmqc8P03oip9smINgKvL828J1SyfNsef8w
zVwWJGb66ADTsG1U7ZYy+TgrLPX21l8p6d5oW+cj28A80WuXQj3usP4D/3XewEbyFuCTED/qBHmiS6Sd
VIdaVYLrMOJErbLLOK6smzSwyu+AGZ/ONHGpZwy5aU8VDjsVH6aPiZSNn5ZTG7AWmbZOS4dk/cQ0LUsP
N79bwDL2B66kOB15FDnXl7iRiVylF0p3N+QCMs14Ul6e/xknpH1v7kRd0uJl+/V7ZVRdquBnzhQsrDeo
VfOwZWpT3H45DszfbvA3AAD//0JyEpx5DAAA
`,
	},

	"/image-layout-schema.json": {
		local:   "image-layout-schema.json",
		size:    414,
		modtime: 1480548955,
		compressed: `
H4sIAAAJbogA/2yPwUrEMBCG732KIXq0TQVPue5pQdiD4EU8xHa2zWKTOJkKi/TdJZlWD91TmD/z8c3/
UwGoHlNHLrILXhlQp4j+EDxb55HgONkB4dlew8zw0o04WfWQqfskgwE1Mkej9SUFX0vaBBp0T/bMdfuk
JbsTzvUbkozWIaLvNlkqmGxrl8X1ZxELydeImQ0fF+zWLFKISOwwKQO5TTZkUi5+RUpSS/72bb9lA8IZ
eEQ4HY6wMxdusycm54f/HP08KQNv6wygHpu2adU6v5d3qQCWcjDh1+wI+z/k1rlV5pbqNwAA//8bwMuB
ngEAAA==
`,
	},

	"/image-manifest-schema.json": {
		local:   "image-manifest-schema.json",
		size:    1139,
		modtime: 1480556516,
		compressed: `
H4sIAAAJbogA/6RSsW7bMBDd/RUHJlsjsSk6ae2UoejQoEvRgRVP8gXSkeUxRg3D/16QFCMr7lA0o574
3r137047AGVR+kA+kmPVgfrikT85joYYAzzMZkT4bJgGlAhfPfY0UG/y67tEv5V+j7NJ1H2MvtP6SRw3
BW1dGLUNZojN+4+6YDeFR7ZSpNPaeeS+TpVMK681JQd6XhwUbjx6TGz38wn7BfPBeQyRUFQHKRiAKhLf
MEgJV+DrzI97EhgIJwtSEqJA3CPk4VCHQ9GDQxEEI2AYiCOOGLKLLP5/yfTW7ItazXo1Ziam+XlWHXxY
MfO7Yhk6lz9qRkvmsSid3uhz1bryKDEQjyuOnL18X74BlPF+Ws5HH9i2rqc267dVvz3cv0sXpBbSj02Q
3vFA42WK24BDmp28I8emVutCu8pU+mSOGOSSXq2bEMxxs9yHiHN6er8WuyCnNc8/TQc4b1wYZhfzDuTt
fTQTSdSXknevV2NxkKZsOZm60RYHYsqvN8xd9ZqdqoC/nimgfalQ/e1G1euDqCVtd75LVZ53fwIAAP//
E6WkmXMEAAA=
`,
	},

	"/manifest-list-schema.json": {
		local:   "manifest-list-schema.json",
		size:    1101,
		modtime: 1479372487,
		compressed: `
H4sIAAAJbogA/6ySMY8TMRCF+/yKkY8OsgZEtS00SCAKTjSIwqxns3OKx8bjizid8t+R7fVml1Ag3ZV5
8bz3vZ153AEoizJECok8qx7Ul4D83nMyxBjhozMHhM+GaURJ8IkkwdeAA400mDLyKnu8kGFCZ/L8lFLo
tb4Tz/uqdj4etI1mTPvX73TVbuoc2TYivdY+IA8tWspYfa0pY2g3Y+yPJKkapIeA2cL/vMNh1kL0AWMi
FNVDrgigqs83jFJrVvm6/e1EAiPh0YLUmiiQJoRCABsCqKZwqq5gBAwDccIDxoJSEp7QUW+xF8vW+irL
EZO7d6qHtxfN/G5akc71H+XQkrmtTo/PAXsxvAKVFIkPFx25AH2ffwMoE8JxPil9Ytv5gboS0rWQLod0
pzcv82mpefLHttL8VNaVGoOJ0Tys1pLQrd/lK4445pcWR9nX8Jx1oy2OxJTZZOn8YT4cHxvLecNimH0q
feS5PvDacunxX9DryV1jLaQq4q97imiXdah/HZ36e7mrb73Lazjv/gQAAP//o026600EAAA=
`,
	},

	"/": {
		isDir: true,
		local: "/",
	},
}
