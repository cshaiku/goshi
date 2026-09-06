"""Exercise each checked-in installer against isolated fake-build fixtures."""
import subprocess,tempfile,os
from pathlib import Path
repos=[str(Path(__file__).resolve().parents[1])]
for source in repos:
 with tempfile.TemporaryDirectory(prefix='sdf install check ') as tmp:
  root=Path(tmp);repo=root/'source';(repo/'scripts').mkdir(parents=True)
  script=Path(source)/'scripts/build-local.sh';(repo/'scripts/build-local.sh').write_text(script.read_text())
  (repo/'patterns').mkdir();(repo/'patterns/a.pattern.json').write_text('{}')
  (repo/'goshi.self.model.yaml').write_text('fixture')
  fake=root/'tools';fake.mkdir()
  go=fake/'go';go.write_text('''#!/bin/sh
case "$1" in
 env) case "$2" in GOOS) echo "${GOOS:-darwin}";; GOHOSTOS) echo darwin;; *) echo arm64;; esac;;
 version) echo 'go version fixture';;
 build) test "${FAIL_BUILD:-0}" = 0 || exit 9; printf '#!/bin/sh\\nexit 0\\n' > "$4";;
esac
''');go.chmod(0o755)
  git=fake/'git';git.write_text('#!/bin/sh\nif [ "$1" = rev-parse ]; then echo fixture; fi\n');git.chmod(0o755)
  prefix=root/'prefix with spaces'
  env=dict(os.environ,GO=str(go),PREFIX=str(prefix),PATH=str(fake)+':'+os.environ['PATH'])
  def run(extra=None,args=()):
   return subprocess.run(['bash','scripts/build-local.sh',*args],cwd=repo,env=env|dict(extra or {}),capture_output=True,text=True)
  assert run(args=['--compile-only']).returncode==0
  assert not prefix.exists()
  r=run();assert r.returncode==0,r.stderr
  files=list((prefix/'bin').iterdir());assert len(files)==1
  binary=files[0];original=binary.read_bytes()
  assert run({'FAIL_BUILD':'1'}).returncode!=0
  assert binary.read_bytes()==original
  assert run({'GOOS':'linux'}).returncode!=0
  assert binary.read_bytes()==original
  assert run().returncode==0
  if "tool_name='loom'" in script.read_text():
   patterns=prefix/'share/loom/patterns';assert patterns.is_symlink()
   assert (patterns/'a.pattern.json').is_file()
   assert not (patterns/'patterns-link').exists()
  print(source+': compile-only, install, repeated install, failed build preservation, and cross-compile refusal passed')
