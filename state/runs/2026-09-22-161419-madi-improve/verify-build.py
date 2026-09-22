import pathlib, subprocess, sys, unittest
root = pathlib.Path(sys.argv.pop(1)).resolve()
mode = sys.argv.pop(1)
class BuildContract(unittest.TestCase):
    def run_command(self, *args, cwd=root):
        result = subprocess.run(args, cwd=cwd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        print(f'{args}: exit {result.returncode}\n{result.stdout}', flush=True)
        self.assertEqual(result.returncode, 0)
    def test_go_compiles_and_vets(self):
        self.run_command('go', 'build', './...')
        self.run_command('go', 'vet', './...')
    def test_vite_preserves_empty_placeholder_and_real_assets(self):
        self.run_command('npm', 'run', 'build', cwd=root / 'web')
        dist = root / 'web/dist'
        self.assertTrue((dist / '.gitkeep').is_file(), 'Vite deleted the tracked placeholder')
        self.assertEqual((dist / '.gitkeep').read_bytes(), b'')
        self.assertTrue((dist / 'index.html').is_file())
        self.assertTrue(any((dist / 'assets').glob('*.js')))
        self.assertNotIn('.gitkeep', (dist / 'sw.js').read_text())
        self.run_command('git', 'diff', '--exit-code', '--', 'web/dist')
        self.run_command('go', 'build', './...')
        self.run_command('go', 'vet', './...')
suite = unittest.TestSuite([BuildContract('test_go_compiles_and_vets')])
if mode == 'web':
    suite.addTest(BuildContract('test_vite_preserves_empty_placeholder_and_real_assets'))
sys.exit(not unittest.TextTestRunner(verbosity=2).run(suite).wasSuccessful())
