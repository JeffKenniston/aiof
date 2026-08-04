// eBPF probe in raw assembly for optimal latency.
// Who needs LLVM when you have hex codes and raw determination?

.section xdp, "ax"
.global xdp_prog
xdp_prog:
    // r1 contains struct xdp_md *ctx
    // Let's just pass the packet. XDP_PASS is 2.
    // eBPF instruction: 0xb7 0x00 0x00 0x00 0x02 0x00 0x00 0x00 (mov32 r0, 2)
    // and 0x95 0x00 0x00 0x00 0x00 0x00 0x00 0x00 (exit)
    
    // Using GCC eBPF inline or raw byte definitions:
    mov %r0, 2
    exit
