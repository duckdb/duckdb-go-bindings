#!/usr/bin/env python3
"""Fit static archives under GitHub's file limit without discarding exports.

Run on the downloaded lib/<platform> directory before committing it. Consumers
need no packaging tools: Linux uses ordinary archives, and split archives are
resolved by GNU ld's GROUP command through the original library name.
"""

import argparse
import collections
import os
import subprocess
import tempfile
from pathlib import Path


MAX_FILE_SIZE = 100 * 1024 * 1024
AR = os.environ.get("LLVM_AR", "llvm-ar")
NM = os.environ.get("LLVM_NM", "llvm-nm")
LLD = os.environ.get("LD_LLD", "ld.lld")

# Keep relocations, constructors, unwind information, and every exported symbol.
# data.rel.ro must stay separate from writable data, as must the TLS sections.
MERGE_SECTIONS = """SECTIONS {
    .text : { *(.text .text.*) }
    .rodata : { *(.rodata .rodata.*) }
    .data.rel.ro : { *(.data.rel.ro .data.rel.ro.*) }
    .data : { *(.data .data.*) }
    .bss : { *(.bss .bss.*) }
    .gcc_except_table : { *(.gcc_except_table .gcc_except_table.*) }
    .tdata : { *(.tdata .tdata.*) }
    .tbss : { *(.tbss .tbss.*) }
}
"""


def defined_symbols(paths):
    output = subprocess.check_output(
        [NM, "--defined-only", "--extern-only", "--format=posix", *paths], text=True
    )
    return collections.Counter(
        tuple(fields[:2])
        for line in output.splitlines()
        if len(fields := line.split()) >= 2
    )


def compact_elf(source, output):
    script = output.with_suffix(".ld")
    script.write_text(MERGE_SECTIONS)
    subprocess.run(
        [LLD, "-r", "--force-group-allocation", "-T", script, source, "-o", output],
        check=True,
    )
    if defined_symbols([source]) != defined_symbols([output]):
        raise ValueError(f"Compaction changed exported symbols in {source}")


def extract_members(archive, directory):
    # Extract by ordinal, not filename: ar permits multiple members with the
    # same name, which 'ar x' would silently overwrite. Ignore archive indexes
    # and name tables; llvm-ar regenerates them when writing each output part.
    members = []
    with archive.open("rb") as stream:
        if stream.read(8) != b"!<arch>\n":
            raise ValueError(f"Expected an ordinary archive: {archive}")
        while header := stream.read(60):
            if len(header) != 60 or header[58:] != b"`\n":
                raise ValueError(f"Invalid archive header in {archive}")
            name = header[:16].decode("ascii").strip()
            size = int(header[48:58])
            data = stream.read(size)
            if len(data) != size:
                raise ValueError(f"Truncated archive member in {archive}")
            if size % 2:
                stream.read(1)
            if name.startswith("#1/"):
                name_size = int(name[3:])
                name = data[:name_size].decode("ascii").rstrip("\0")
                data = data[name_size:]
            if name in ("/", "//", "/SYM64/") or name.startswith("__.SYMDEF"):
                continue
            member = directory / f"member{len(members):04d}.o"
            member.write_bytes(data)
            members.append(member)
    if not members:
        raise ValueError(f"No object members in {archive}")
    return members


def package_archive(archive, platform, max_file_size=MAX_FILE_SIZE):
    original_size = archive.stat().st_size
    if original_size <= max_file_size:
        return
    if not platform.startswith(("linux-", "windows-")):
        raise ValueError(f"Cannot package oversized archive for {platform}: {archive}")

    with tempfile.TemporaryDirectory(prefix="package-libs-", dir=archive.parent) as temp:
        work = Path(temp)
        members = extract_members(archive, work)
        if platform.startswith("linux-"):
            for i, member in enumerate(members):
                # DuckDB's Linux LTO archive has a single oversized ELF object.
                # Multi-object archives can be split without changing objects.
                if len(members) == 1 or member.stat().st_size > max_file_size:
                    compact = work / f"compact{i:04d}.o"
                    compact_elf(member, compact)
                    members[i] = compact

        parts = []

        def write_parts(objects):
            part = work / f"{archive.stem}_part{len(parts) + 1}.a"
            subprocess.run([AR, "rcsD", part, *objects], check=True)
            if part.stat().st_size <= max_file_size:
                parts.append(part)
                return
            part.unlink()
            if len(objects) == 1:
                raise ValueError(f"A single object still exceeds the file limit in {archive}")
            # Balance bytes while retaining the original object order.
            sizes = [obj.stat().st_size for obj in objects]
            target = sum(sizes) / 2
            accumulated = 0
            midpoint = 1
            for i, size in enumerate(sizes[:-1], 1):
                accumulated += size
                midpoint = i
                if accumulated >= target:
                    break
            write_parts(objects[:midpoint])
            write_parts(objects[midpoint:])

        write_parts(members)
        if defined_symbols([archive]) != defined_symbols(parts):
            raise ValueError(f"Packaging changed exported symbols in {archive}")

        if len(parts) == 1:
            parts[0].replace(archive)
            print(f"{archive}: {original_size:,} -> {archive.stat().st_size:,} bytes")
        else:
            libraries = " ".join("-l" + p.stem.removeprefix("lib") for p in parts)
            script = work / archive.name
            script.write_text(
                "/* Split archive; retain whole objects and rescan dependencies. */\n"
                f"GROUP ( {libraries} )\n"
            )
            for part in parts:
                destination = archive.parent / part.name
                part.replace(destination)
                print(f"{destination}: {destination.stat().st_size:,} bytes")
            script.replace(archive)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path, help="downloaded lib/<platform> directory")
    args = parser.parse_args()
    directory = args.directory.resolve()
    archives = sorted(directory.glob("*.a"))
    if not archives:
        parser.error(f"No archives in {directory}")
    for archive in archives:
        package_archive(archive, directory.name)
    for archive in directory.glob("*.a"):
        if archive.stat().st_size > MAX_FILE_SIZE:
            raise ValueError(f"Archive still exceeds the file limit: {archive}")


if __name__ == "__main__":
    main()
