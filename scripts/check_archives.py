"""Validate the exact six archives and checksums the release publishes."""
import argparse
import hashlib
from pathlib import Path
import platform
import subprocess
import tarfile
import tempfile
import zipfile
import os

MATRIX = [(system, arch, 'zip' if system == 'Windows' else 'tar.gz')
          for system in ('Linux', 'Darwin', 'Windows') for arch in ('x86_64', 'arm64')]


def host_archive():
    arch = {'amd64': 'x86_64', 'aarch64': 'arm64'}.get(platform.machine().lower(), platform.machine().lower())
    system = platform.system()
    return Path('dist') / f'gdlint_{system}_{arch}.{ "zip" if system == "Windows" else "tar.gz" }'


def extract(archive, directory):
    binary = 'gdlint.exe' if archive.suffix == '.zip' else 'gdlint'
    expected = {binary, 'README.md'}
    if archive.suffix == '.zip':
        with zipfile.ZipFile(archive) as package:
            if set(package.namelist()) != expected or len(package.infolist()) != 2:
                raise ValueError('unexpected ZIP contents')
            package.extractall(directory)
    else:
        with tarfile.open(archive) as package:
            members = package.getmembers()
            if {item.name for item in members} != expected or len(members) != 2 or not all(item.isfile() for item in members):
                raise ValueError('unexpected TAR contents')
            package.extractall(directory, filter='data')
    return Path(directory) / binary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--record', action='store_true')
    args = parser.parse_args()
    expected = {f'gdlint_{system}_{arch}.{extension}' for system, arch, extension in MATRIX}
    actual = {p.name for p in Path('dist').glob('*') if p.name.endswith(('.tar.gz', '.zip'))}
    if actual != expected:
        raise ValueError('release must contain exactly all six supported archives')
    checksums = ''.join(hashlib.sha256((Path('dist') / name).read_bytes()).hexdigest()+'  '+name+'\n' for name in sorted(expected))
    manifest = Path('dist/checksums.txt')
    if args.record:
        manifest.write_text(checksums)
        return
    if manifest.read_text() != checksums:
        raise ValueError('release checksums differ from archive bytes')
    for name in sorted(expected):
        with tempfile.TemporaryDirectory() as directory:
            extract(Path('dist') / name, directory)
    with tempfile.TemporaryDirectory() as directory:
        binary = extract(host_archive(), directory)
        version = subprocess.check_output([binary, 'version'], text=True, timeout=10).strip()
        wanted = os.environ.get('VERSION', 'dev')
        if not version.startswith(f'gdlint {wanted} ('):
            raise ValueError('archived binary version differs from the requested release')
        print(version)
    print('All six archives and their checksums verified')


if __name__ == '__main__':
    main()
