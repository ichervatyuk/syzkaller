// Copyright 2015 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"bytes"
	"sort"
	"text/template"

	. "github.com/google/syzkaller/sysparser"
)

type Arch struct {
	Name  string
	CARCH []string
}

var archs = []*Arch{
	{"amd64", []string{"__x86_64__"}},
	{"arm64", []string{"__aarch64__"}},
	{"ppc64le", []string{"__ppc64__", "__PPC64__", "__powerpc64__"}},
}

var syzkalls = map[string]uint64{
	"syz_test":            1000001,
	"syz_open_dev":        1000002,
	"syz_open_pts":        1000003,
	"syz_fuse_mount":      1000004,
	"syz_fuseblk_mount":   1000005,
	"syz_emit_ethernet":   1000006,
	"syz_kvm_setup_cpu":   1000007,
	"syz_extract_tcp_res": 1000008,
}

func generateExecutorSyscalls(syscalls []Syscall, consts map[string]map[string]uint64) {
	var data SyscallsData
	for _, arch := range archs {
		var calls []SyscallData
		for _, c := range syscalls {
			syscallNR := -1
			if nr, ok := consts[arch.Name]["__NR_"+c.CallName]; ok {
				syscallNR = int(nr)
			}
			calls = append(calls, SyscallData{c.Name, syscallNR})
		}
		data.Archs = append(data.Archs, ArchData{arch.CARCH, calls})
	}
	for name, nr := range syzkalls {
		data.FakeCalls = append(data.FakeCalls, SyscallData{name, int(nr)})
	}
	sort.Sort(SyscallArray(data.FakeCalls))

	hdrcode := "executor/syscalls.h"
	logf(1, "Generate header with syscall numbers in %v", hdrcode)
	buf := new(bytes.Buffer)
	if err := syscallsTempl.Execute(buf, data); err != nil {
		failf("failed to execute syscalls template: %v", err)
	}
	writeFile(hdrcode, buf.Bytes())
}

type SyscallsData struct {
	Archs     []ArchData
	FakeCalls []SyscallData
}

type ArchData struct {
	CARCH []string
	Calls []SyscallData
}

type SyscallData struct {
	Name string
	NR   int
}

type SyscallArray []SyscallData

func (a SyscallArray) Len() int           { return len(a) }
func (a SyscallArray) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a SyscallArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

var syscallsTempl = template.Must(template.New("").Parse(
	`// AUTOGENERATED FILE

{{range $c := $.FakeCalls}}#define __NR_{{$c.Name}}	{{$c.NR}}
{{end}}

struct call_t {
	const char*	name;
	int		sys_nr;
};

{{range $arch := $.Archs}}
#if {{range $cdef := $arch.CARCH}}defined({{$cdef}}) || {{end}}0
call_t syscalls[] = {
{{range $c := $arch.Calls}}	{"{{$c.Name}}", {{$c.NR}}},
{{end}}
};
#endif
{{end}}
`))
