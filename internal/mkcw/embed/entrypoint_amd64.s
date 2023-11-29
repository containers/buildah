	.section	.rodata.1,"aMS",@progbits,1
msg:
	.string	"This image is designed to be run as a confidential workload using libkrun.\n"
	.section	.text._start,"ax",@progbits
	.globl	_start
	.type	_start,@function
_start:
	movq	$1, %rax	# write
	movq	$2, %rdi	# fd=stderr_fileno
	movq	$msg, %rsi	# message
	movq	$75, %rdx	# length
	syscall
	movq	$60, %rax	# exit
	movq	$1, %rdi	# status=1
	syscall
	.section	.note.GNU-stack,"",@progbits
