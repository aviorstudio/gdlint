"""Exercise the actual archived CLI using disposable project directories."""
import json
from pathlib import Path
import subprocess
import tempfile

from check_archives import extract, host_archive


def run(binary, root, args, success=True, message=None):
    result = subprocess.run([binary, *args], cwd=root, text=True, capture_output=True, timeout=20)
    if (result.returncode == 0) != success:
        raise AssertionError(result.stdout+result.stderr)
    if message and message not in result.stdout+result.stderr:
        raise AssertionError('expected diagnostic missing: '+message)


with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    binary = extract(host_archive().resolve(), root / 'binary')
    project = root / 'project'
    project.mkdir()
    run(binary, project, ['version'], message='gdlint ')
    run(binary, project, ['unknown'], False, 'unknown command')
    run(binary, project, ['init', 'version'], False, 'expected at most one command')
    run(binary, project, [], False, 'project.godot')
    marker = project / 'project.godot'
    marker.mkdir()
    run(binary, project, [], False, 'expected a file')
    marker.rmdir()
    marker.write_text('[application]\nconfig/name="Disposable CLI fixture"\n')
    run(binary, project, ['init'], message='Created default configuration')
    config = project / 'gdlint.json'
    original = config.read_bytes()
    assert isinstance(json.loads(original), dict) and json.loads(original)
    run(binary, project, ['init'], False, 'config already exists')
    assert config.read_bytes() == original
    run(binary, project, [])
    config.write_text('{malformed')
    run(binary, project, [], False, 'invalid config file')
print('Nine archived CLI checks passed, including restored success and config preservation')
