"""Run the CLI against disposable Godot projects, including destructive fixes."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

BINARY=Path(os.environ.get('GDLINT_TEST_BINARY',str(Path(os.environ.get('CARGO_TARGET_DIR','target'))/'debug'/('gdlint.exe' if os.name=='nt' else 'gdlint')))).resolve()
ANSI=re.compile(r'\x1b\[[0-9;]*m')


class Project(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root=Path(self.temp.name)
        self.put('project.godot','[application]\nconfig/name="Disposable lint fixture"\n')
    def put(self,name,text):
        path=self.root/name
        path.parent.mkdir(parents=True,exist_ok=True)
        path.write_text(text)
        return path
    def config(self,errors=None,warnings=None,settings=None):
        config={}
        if errors is not None:
            config['errors']={key:False for key in ['unused_functions','unused_signals','unused_constants','unused_enums','unused_class_names','unused_files','print_statements','comments','pass_statements','orphaned_uids','indentation','has_method','variant_usage','missing_return_types']}
            config['errors'].update(errors)
        if warnings is not None: config['warnings']=warnings
        if settings is not None: config['settings']=settings
        self.put('gdlint.json',json.dumps(config))
    def cli(self,*args,code=None):
        result=subprocess.run([str(BINARY),*args],cwd=self.root,capture_output=True,text=True,timeout=15)
        result.stdout=ANSI.sub('',result.stdout)
        if code is not None: self.assertEqual(result.returncode,code,result.stdout+result.stderr)
        return result


class CLI(Project):
    def test_project_commands_and_init_preserves_existing_config(self):
        (self.root/'project.godot').unlink()
        self.assertIn('gdlint ',self.cli('version',code=0).stdout)
        self.assertIn('project.godot',self.cli(code=1).stderr)
        (self.root/'project.godot').mkdir()
        self.assertIn('expected a file',self.cli(code=1).stderr)
        (self.root/'project.godot').rmdir()
        self.put('project.godot','')
        self.assertIn('Created default',self.cli('init',code=0).stdout)
        config=(self.root/'gdlint.json').read_bytes()
        self.assertEqual(config,(Path(__file__).parents[1]/'src/defaults.json').read_bytes())
        self.assertIn('config already exists',self.cli('init',code=1).stderr)
        self.assertEqual(config,(self.root/'gdlint.json').read_bytes())
        self.assertIn('unknown command',self.cli('unknown',code=1).stderr)
        self.assertIn('expected at most one command',self.cli('init','version',code=1).stderr)
    def test_defaults_find_print_indent_and_orphan_but_preserve_optional_rules(self):
        self.put('one.gd','# ordinary comment\nclass_name One\nsignal unused\nfunc helper():\n    print("hello")\n')
        self.put('missing.gd.uid','uid://orphan')
        output=self.cli(code=1).stdout
        for label in ['Print statements: 1','Indentation issues: 1','Orphaned UID files: 1']:
            self.assertIn(label,output)
        for label in ['Unused signals:','Unused functions:','Comments to remove:','Missing return type annotations:']:
            self.assertNotIn(label,output)
    def test_engine_callbacks_getters_setters_and_rpc_are_protected(self):
        self.config(dict(unused_functions=True))
        names=['_get_configuration_warnings','_get_property_list','_property_can_revert','_property_get_revert','_get','_set','_to_string','_ready','_on_pressed','get_value','set_value']
        text='class_name Callbacks\n'+''.join(f'func {name}():\n\tpass\n' for name in names)
        text+='@rpc("any_peer")\nfunc remote():\n\tpass\nfunc _helper():\n\tpass\nfunc _get_configuration_warnings_extra():\n\tpass\n'
        self.put('callbacks.gd',text)
        output=self.cli(code=1).stdout
        self.assertIn('Unused functions: 2',output)
        self.assertIn(': _helper',output)
        self.assertIn(': _get_configuration_warnings_extra',output)
        self.assertNotIn(': remote\n',output)
    def test_calls_connect_callbacks_callable_and_string_references(self):
        self.config(dict(unused_functions=True))
        self.put('calls.gd','class_name Calls\nsignal done\nfunc _ready():\n\tused()\n\tdone.connect(_callback)\n\tvar action = [_callable]\n\tvar text = "dynamic"\nfunc used():\n\tpass\nfunc _callback():\n\tpass\nfunc _callable():\n\tpass\nfunc dynamic():\n\tpass\nfunc unused():\n\tpass\n')
        output=self.cli(code=1).stdout
        self.assertIn('Unused functions: 1',output)
        self.assertIn(': unused',output)
    def test_signals_constants_enums_and_enum_values_follow_existing_usage_rules(self):
        self.config(dict(unused_signals=True,unused_constants=True,unused_enums=True,unused_class_names=True))
        self.put('entities.gd','class_name Entities\nsignal untouched\nsignal emitted\nconst KEEP = 1\nenum State {\n\tIDLE,\n\tBUSY\n}\nenum Never {\n\tONE,\n}\nfunc _ready():\n\temitted.emit()\n\tvar state = State.IDLE\n')
        output=self.cli(code=1).stdout
        self.assertIn('Unused signals: 1',output)
        self.assertIn(': untouched',output)
        self.assertIn('Unused enums: 1',output)
        self.assertIn(': Never',output)
        self.assertNotIn('Unused constants:',output)
        self.assertNotIn('Unused class names:',output)
    def test_file_references_scenes_autoloads_class_names_and_build_paths(self):
        self.config(dict(unused_files=True))
        self.put('project.godot','[autoload]\nGlobal="*res://global.gd"\n')
        for name in ['global.gd','scene.gd','preloaded.gd','loaded.gd','parent.gd','unused.gd','build/generated.gd','src/routes/build/live.gd']:
            self.put(name,'extends Node\n')
        self.put('class.gd','class_name Widget\n')
        self.put('main.gd','extends "res://parent.gd"\nconst PRELOAD = preload("res://preloaded.gd")\nfunc _ready():\n\tvar loaded = load("./loaded.gd")\n\tvar widget: Widget = null\n')
        self.put('main.tscn','[ext_resource type="Script" path="res://main.gd" id="1"]\nscript = "res://scene.gd"\n')
        output=self.cli(code=1).stdout
        self.assertIn('Unused files: 2',output)
        self.assertIn('• src/routes/build/live.gd',output)
        self.assertIn('• unused.gd',output)
        self.assertNotIn('generated.gd',output)
        self.assertNotIn('• global.gd',output)
        self.assertNotIn('• class.gd',output)
    def test_ignore_patterns_have_separate_general_unused_and_variant_scopes(self):
        self.config(dict(unused_functions=True,variant_usage=True,print_statements=True),settings=dict(ignore_patterns=['skip*.gd'],unused_ignore_patterns=['unused/'],variant_ignore_patterns=['variant/']))
        for index,name in enumerate(['skip-me.gd','unused/a.gd','variant/a.gd','active.gd']):
            self.put(name,f'class_name Example{index}\nfunc helper{index}(value: Variant):\n\tprint(value)\n')
        output=self.cli(code=1).stdout
        self.assertIn('Unused functions: 2',output)
        self.assertIn('Print statements: 3',output)
        self.assertIn('Explicit Variant usages (disallowed once enabled): 2',output)
        self.assertNotIn('skip-me.gd',output)
    def test_comments_and_ignore_print_on_current_or_previous_line(self):
        self.config(dict(print_statements=True,comments=True))
        self.put('comments.gd','# ordinary\n# [TECH DEBT] keep\nfunc _ready():\n\t# gdlint-ignore-print\n\tprint("keep")\n\tprint("also keep") # gdlint-ignore-print\n\tprint("previous line suppresses this")\n\tprint_rich("remove")\n\tDebugLogUtil.log("remove")\n')
        output=self.cli(code=1).stdout
        self.assertIn('Comments to remove: 1',output)
        self.assertIn('Print statements: 2',output)
    def test_pass_only_body_survives_but_extra_pass_is_reported(self):
        self.config(dict(pass_statements=True))
        self.put('passes.gd','func _ready():\n\tpass\nfunc work():\n\tvar value = 1\n\tpass\nfunc later():\n\tpass\n\tvar value = 2\n')
        output=self.cli(code=1).stdout
        self.assertIn('Unnecessary pass statements: 2',output)
        self.assertIn('passes.gd:5',output)
        self.assertIn('passes.gd:7',output)
        self.assertNotIn('passes.gd:2',output)
    def test_explicit_variant_has_method_indent_and_return_annotations(self):
        self.config(dict(variant_usage=True,has_method=True,indentation=True,missing_return_types=True))
        self.put('rules.gd','extends Node\nfunc public(value: Variant):\n    if has_method("work"):\n\t pass\nfunc typed(\n\tvalue: int\n) -> void:\n\tpass\nfunc _private():\n\tpass\n# Variant in a comment\n')
        output=self.cli(code=1).stdout
        for label in ['Explicit Variant usages (disallowed once enabled): 1','❌ has_method() calls (ALWAYS ERRORS): 1','Indentation issues: 2','Missing return type annotations: 1']:
            self.assertIn(label,output)
        self.assertIn("public function 'public' missing return type annotation",output)
    def test_debt_includes_ignored_and_build_scripts_and_is_read_only(self):
        self.config(dict(),settings=dict(ignore_patterns=['skip/']))
        files={'skip/ignored.gd':'# [TECH DEBT] ignored debt\n','build/generated.gd':'\t# [TECH DEBT] generated debt\n','normal.gd':'# other\n# [TECH DEBT] ordinary debt\n'}
        for name,text in files.items(): self.put(name,text)
        output=self.cli('-debt',code=0).stdout
        self.assertIn('Found 3 [TECH DEBT] comments:',output)
        self.assertIn('skip/ignored.gd:1: ignored debt',output)
        self.assertIn('build/generated.gd:1: generated debt',output)
        for name,text in files.items(): self.assertEqual((self.root/name).read_text(),text)
    def test_config_partial_null_unknown_and_invalid_types(self):
        self.put('gdlint.json','{"errors":{"print_statements":null},"warnings":null,"settings":{"ignore_patterns":null},"unknown":17}')
        self.put('print.gd','func _ready():\n\tprint(1)\n')
        self.assertIn('Print statements: 1',self.cli(code=1).stdout)
        for value in ['{malformed','[]','{"errors":{"print_statements":"yes"}}','{"settings":{"ignore_patterns":[1]}}']:
            self.put('gdlint.json',value)
            self.assertIn('invalid config file',self.cli(code=1).stderr)
        self.put('gdlint.json','{"errors":{"print_statements":false}}')
        self.assertIn('codebase is clean',self.cli(code=0).stdout)
    def test_warnings_are_counted_when_hidden_and_detailed_when_requested(self):
        self.config(dict(),dict(single_use_functions=True,empty_functions=True))
        self.put('warnings.gd','class_name Warnings\nfunc empty():\n\tpass\n')
        output=self.cli(code=0).stdout
        self.assertIn('2 warnings found',output)
        self.assertIn('run with -warn',output)
        self.assertNotIn('=== WARNINGS ===',output)
        output=self.cli('-warn',code=0).stdout
        self.assertIn('Single-use functions: 1',output)
        self.assertIn('Empty functions: 1',output)
    def test_fix_simple_print_comment_pass_and_uid(self):
        self.config(dict(print_statements=True,comments=True,pass_statements=True,orphaned_uids=True))
        self.put('simple.gd','# remove\nfunc _ready():\n\tprint(1)\n\nfunc work():\n\tvar keep = 1\n\tpass\n')
        self.put('gone.gd.uid','uid://old')
        self.cli('-fix',code=0)
        self.assertEqual((self.root/'simple.gd').read_text(),'func _ready():\n\tpass\n\nfunc work():\n\tvar keep = 1')
        self.assertFalse((self.root/'gone.gd.uid').exists())
        self.cli(code=0)
    def test_fix_unused_files_respects_references(self):
        self.config(dict(unused_files=True))
        self.put('keep.gd','extends Node\n')
        self.put('unused.gd','extends Node\n')
        self.put('main.tscn','script = "res://keep.gd"\n')
        self.cli('-fix',code=0)
        self.assertTrue((self.root/'keep.gd').exists())
        self.assertFalse((self.root/'unused.gd').exists())
    def test_fix_unhandled_rules_stops_at_ten_iterations(self):
        self.config(dict(has_method=True))
        script=self.put('unhandled.gd','func _ready():\n\thas_method("work")\n')
        original=script.read_bytes()
        result=self.cli('-fix',code=0)
        self.assertEqual(result.stdout.count('=== Fix iteration'),10)
        self.assertIn('Reached maximum iterations (10)',result.stderr)
        self.assertEqual(script.read_bytes(),original)
    def test_benchmark_and_flag_forms(self):
        self.assertRegex(self.cli('--benchmark=true',code=0).stdout,r'Execution time: \d+\.\d{2}s')
        self.assertNotIn('Execution time:',self.cli('-benchmark=false',code=0).stdout)
        self.assertNotEqual(self.cli('-unknown').returncode,0)
        self.assertNotEqual(self.cli('version','-warn').returncode,0)


class SafeFix(Project):
    def test_contained_print_and_comment_fixes_do_not_shift_function_ranges(self):
        self.config(dict(unused_functions=True,print_statements=True,comments=True))
        self.put('neighbors.gd','class_name Neighbors\nfunc unused():\n\t# remove within unused body\n\tprint("unused")\n\tvar gone = 1\n\nfunc _ready():\n\tvar keep_neighbor = 2\n')
        self.cli('-fix',code=0)
        text=(self.root/'neighbors.gd').read_text()
        self.assertNotIn('func unused',text)
        self.assertNotIn('gone',text)
        self.assertIn('func _ready():\n\tvar keep_neighbor = 2',text)
    def test_single_line_enum_does_not_consume_a_later_dictionary_or_function(self):
        self.config(dict(unused_enums=True))
        self.put('enum.gd','class_name Example\nenum State {IDLE, BUSY}\nfunc _ready():\n\tvar keep = {"hello": "world"}\n\tkeep.get("hello")\n')
        self.cli('-fix',code=0)
        text=(self.root/'enum.gd').read_text()
        self.assertNotIn('enum State',text)
        self.assertIn('func _ready()',text)
        self.assertIn('var keep = {"hello": "world"}',text)
    @unittest.skipIf(os.name=='nt','creating symlinks requires Windows developer privileges')
    def test_symlink_fix_fails_before_modifying_any_file(self):
        self.config(dict(print_statements=True))
        first=self.put('a.gd','func _ready():\n\tprint("first")\n')
        with tempfile.TemporaryDirectory() as outside:
            target=Path(outside)/'outside.gd'
            target.write_text('func _ready():\n\tprint("outside")\n')
            (self.root/'z.gd').symlink_to(target)
            result=self.cli('-fix',code=1)
            self.assertIn('refusing to modify symlink',result.stderr)
            self.assertIn('print("outside")',target.read_text())
            self.assertIn('print("first")',first.read_text())


if __name__=='__main__': unittest.main(verbosity=2)
