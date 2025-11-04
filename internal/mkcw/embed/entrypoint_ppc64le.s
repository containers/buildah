DATA msg+0(SB)/75, $"This image is designed to be run as a confidential workload using libkrun.\n"

GLOBL msg(SB),8,$75

TEXT _start(SB),8-0,$0
	MOVD	$4, R0		// syscall=write
	MOVD	$2, R3		// descriptor=2
	MOVD	$msg(SB), R4	// buffer (msg) address
	MOVD	$75, R5		// buffer (msg) length
	SYSCALL
	MOVD	$1, R0		// syscall=exit
	MOVD	$1, R3		// status=1
	SYSCALL
