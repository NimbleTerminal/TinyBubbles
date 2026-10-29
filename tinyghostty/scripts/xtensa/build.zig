//! Builds libghostty-vt for the ESP32-S3 with the Xtensa Zig fork.
//! scripts/build-libghostty-vt.sh links ./ghostty to a Ghostty checkout.
//! Ghostty's own build cannot change two settings the fork needs:
//!   - Symbols: when Zig strips, the fork's literal pools name internal
//!     functions that the object never defines.
//!   - Function sections: L32R reaches literals at most 256 KiB back, so each
//!     function keeps its literal pool beside it.
const std = @import("std");

pub fn build(b: *std.Build) void {
    const lib = b.dependency("ghostty", .{
        .target = b.standardTargetOptions(.{}),
        .optimize = b.standardOptimizeOption(.{}),
        .@"emit-lib-vt" = true,
        .simd = false,
    }).artifact("ghostty-vt-static");
    keepSymbols(lib.root_module);
    lib.link_function_sections = true;
    lib.link_data_sections = true;
    b.installArtifact(lib);
}

fn keepSymbols(m: *std.Build.Module) void {
    if (m.strip == false) return;
    m.strip = false;
    for (m.import_table.values()) |dep| keepSymbols(dep);
}
