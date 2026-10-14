import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

import package_static_libs as packaging


class PackageStaticLibsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def compile(self, directory, source):
        directory.mkdir(exist_ok=True)
        src = directory / "same.c"
        obj = directory / "same.o"
        src.write_text(source)
        subprocess.run(
            ["cc", "-ffunction-sections", "-fdata-sections", "-c", src, "-o", obj],
            check=True,
        )
        return obj

    def test_small_archive_is_unchanged(self):
        obj = self.compile(self.root / "a", "int answer(void) { return 42; }")
        archive = self.root / "libsmall.a"
        subprocess.run([packaging.AR, "rcsD", archive, obj], check=True)
        original = archive.read_bytes()
        packaging.package_archive(archive, "linux-amd64")
        self.assertEqual(original, archive.read_bytes())

    def test_split_preserves_duplicate_members_and_rescans_dependencies(self):
        # final is needed only after entry -> middle has pulled later parts in.
        # The two same.o members must both survive extraction and repacking.
        final = self.compile(
            self.root / "a",
            "char pad_a[4096] = {1}; int final(void) { return 42; }",
        )
        entry = self.compile(
            self.root / "b",
            "char pad_b[4096] = {1}; int middle(void); "
            "int entry(void) { return middle(); }",
        )
        middle = self.compile(
            self.root / "c",
            "char pad_c[4096] = {1}; int final(void); "
            "int middle(void) { return final(); }",
        )
        archive = self.root / "libexample.a"
        subprocess.run([packaging.AR, "qcD", archive, final, entry, middle], check=True)
        subprocess.run([packaging.AR, "sD", archive], check=True)
        original_symbols = packaging.defined_symbols([archive])
        packaging.package_archive(archive, "windows-amd64", max_file_size=8000)
        parts = sorted(self.root.glob("libexample_part*.a"))
        self.assertGreater(len(parts), 1)
        self.assertTrue(all(p.stat().st_size <= 8000 for p in parts))
        self.assertEqual(original_symbols, packaging.defined_symbols(parts))
        self.assertIn("GROUP", archive.read_text())
        main = self.root / "main.c"
        main.write_text("int entry(void); int main(void) { return entry() != 42; }")
        exe = self.root / "example"
        subprocess.run(["cc", main, "-L" + str(self.root), "-lexample", "-o", exe], check=True)
        subprocess.run([exe], check=True)

    def test_compaction_keeps_unreferenced_exports_weak_symbols_and_tls(self):
        obj = self.compile(self.root / "a", """
            __thread int local_value = 7;
            __attribute__((weak)) int weak_api(void) { return local_value; }
            int future_extension_api(void) { return 13; }
            int entry(void) { return weak_api(); }
        """)
        compact = self.root / "compact.o"
        packaging.compact_elf(obj, compact)
        self.assertEqual(packaging.defined_symbols([obj]), packaging.defined_symbols([compact]))
        main = self.root / "main.c"
        main.write_text(
            "int entry(void); int future_extension_api(void); "
            "int main(void) { return entry() != 7 || future_extension_api() != 13; }"
        )
        exe = self.root / "example"
        subprocess.run(["cc", main, compact, "-o", exe], check=True)
        subprocess.run([exe], check=True)

    def test_unsplittable_object_leaves_original_archive_intact(self):
        obj = self.compile(self.root / "a", "char large[4096] = {1};")
        archive = self.root / "liblarge.a"
        subprocess.run([packaging.AR, "rcsD", archive, obj], check=True)
        original = archive.read_bytes()
        with self.assertRaisesRegex(ValueError, "single object"):
            packaging.package_archive(archive, "windows-amd64", max_file_size=1000)
        self.assertEqual(original, archive.read_bytes())
        self.assertFalse(list(self.root.glob("liblarge_part*.a")))

    def test_windows_split_links_with_mingw_lld(self):
        clang = os.environ.get("LLVM_CLANG", "clang")
        objects = []
        for name, source in [
            ("final", "char pad_a[4096] = {1}; int final(void) { return 42; }"),
            ("entry", "char pad_b[4096] = {1}; int middle(void); "
             "int entry(void) { return middle(); }"),
            ("middle", "char pad_c[4096] = {1}; int final(void); "
             "int middle(void) { return final(); }"),
            ("main", "int entry(void); int mainCRTStartup(void) { return entry() != 42; }"),
        ]:
            src = self.root / f"{name}.c"
            obj = self.root / f"{name}.o"
            src.write_text(source)
            subprocess.run(
                [clang, "--target=x86_64-w64-mingw32", "-c", src, "-o", obj],
                check=True,
            )
            objects.append(obj)
        archive = self.root / "libexample.a"
        subprocess.run([packaging.AR, "rcsD", archive, *objects[:3]], check=True)
        prebuilt = self.root / "prebuilt.go"
        prebuilt.write_text("#cgo LDFLAGS: -lexample -L${SRCDIR}\n")
        packaging.package_archive(archive, "windows-amd64", max_file_size=8000)
        flags = prebuilt.read_text().split(": ", 1)[1].split()
        flags = [flag.replace("${SRCDIR}", str(self.root)) for flag in flags]
        subprocess.run(
            [clang, "--target=x86_64-w64-mingw32", "-nostdlib",
             "-fuse-ld=" + shutil.which(packaging.LLD),
             "-Wl,--entry,mainCRTStartup", "-Wl,--subsystem,console",
             objects[3], *flags,
             "-o", self.root / "example.exe"],
            check=True,
        )
        self.assertEqual(
            ["-Wl,--start-group", "-lexample_part1", "-lexample_part2",
             "-lexample_part3", "-Wl,--end-group", "-L${SRCDIR}"],
            prebuilt.read_text().split(": ", 1)[1].split(),
        )

    def test_upstream_fetch_restores_single_archive_linking(self):
        obj = self.compile(self.root / "a", "int answer(void) { return 42; }")
        archive = self.root / "libexample.a"
        subprocess.run([packaging.AR, "rcsD", archive, obj], check=True)
        prebuilt = self.root / "prebuilt.go"
        prebuilt.write_text(
            "#cgo LDFLAGS: -lother -Wl,--start-group -lexample_part1 "
            "-lexample_part2 -lexample_part3 -Wl,--end-group -lm -L${SRCDIR}\n"
        )
        original = archive.read_bytes()
        subprocess.run(
            ["python3", packaging.__file__, "--sync-linker-flags", self.root],
            check=True,
        )
        self.assertEqual(original, archive.read_bytes())
        self.assertEqual(
            "#cgo LDFLAGS: -lother -lexample -lm -L${SRCDIR}\n",
            prebuilt.read_text(),
        )

    def test_repackaging_updates_part_count(self):
        objects = [
            self.compile(self.root / name, f"char pad_{name}[4096] = {{1}};")
            for name in ["a", "b", "c"]
        ]
        archive = self.root / "libexample.a"
        subprocess.run([packaging.AR, "qcD", archive, *objects], check=True)
        subprocess.run([packaging.AR, "sD", archive], check=True)
        prebuilt = self.root / "prebuilt.go"
        prebuilt.write_text("#cgo LDFLAGS: -lother -lexample -lm -L${SRCDIR}\n")
        packaging.package_archive(archive, "windows-amd64", max_file_size=12000)
        self.assertEqual(
            "#cgo LDFLAGS: -lother -Wl,--start-group -lexample_part1 "
            "-lexample_part2 -Wl,--end-group -lm -L${SRCDIR}\n",
            prebuilt.read_text(),
        )

        # Simulate another download with a smaller packaging size budget.
        archive.unlink()
        subprocess.run([packaging.AR, "qcD", archive, *objects], check=True)
        subprocess.run([packaging.AR, "sD", archive], check=True)
        packaging.package_archive(archive, "windows-amd64", max_file_size=8000)
        self.assertEqual(
            "#cgo LDFLAGS: -lother -Wl,--start-group -lexample_part1 "
            "-lexample_part2 -lexample_part3 -Wl,--end-group -lm -L${SRCDIR}\n",
            prebuilt.read_text(),
        )


if __name__ == "__main__":
    unittest.main()
