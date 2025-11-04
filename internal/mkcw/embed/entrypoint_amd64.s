DATA msg+0(SB)/75, $"This image is designed to be run as a confidential workload using libkrun.\n"

GLOBL msg(SB),8,$75

TEXT _start(SB),8-0,$0
	MOVQ	$1, AX		// syscall=write
	MOVQ	$2, DI		// descriptor=2
	MOVQ	$msg(SB), SI	// buffer (msg) address
	MOVQ	$75, DX		// buffer (msg) length
	SYSCALL
	MOVQ	$60, AX		// syscall=exit
	MOVQ	$1, DI		// status=1
	SYSCALL
