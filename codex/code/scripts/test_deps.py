
import io
import os
import sys
import unittest
from unittest.mock import patch, mock_open
from deps import (
    parse_dependencies,
    find_circular_dependencies,
    print_dependency_tree,
    print_reverse_dependencies,
    main
)

class TestDeps(unittest.TestCase):

    def setUp(self):
        self.yaml_content = """
repos:
  core-tenant:
    depends: [core-php]
  core-admin:
    depends: [core-php, core-tenant]
  core-php:
    depends: []
  core-api:
    depends: [core-php]
  core-analytics:
    depends: [core-php, core-api]
"""
        self.graph = {
            'core-tenant': ['core-php'],
            'core-admin': ['core-php', 'core-tenant'],
            'core-php': [],
            'core-api': ['core-php'],
            'core-analytics': ['core-php', 'core-api'],
        }
        self.circular_yaml_content = """
repos:
  module-a:
    depends: [module-b]
  module-b:
    depends: [module-c]
  module-c:
    depends: [module-a]
"""
        self.circular_graph = {
            'module-a': ['module-b'],
            'module-b': ['module-c'],
            'module-c': ['module-a'],
        }

    def test_parse_dependencies(self):
        with patch("builtins.open", mock_open(read_data=self.yaml_content)):
            graph = parse_dependencies("dummy_path.yaml")
            self.assertEqual(graph, self.graph)

    def test_find_circular_dependencies(self):
        cycles = find_circular_dependencies(self.circular_graph)
        self.assertEqual(len(cycles), 1)
        self.assertIn('module-a', cycles[0])
        self.assertIn('module-b', cycles[0])
        self.assertIn('module-c', cycles[0])

    def test_find_no_circular_dependencies(self):
        cycles = find_circular_dependencies(self.graph)
        self.assertEqual(len(cycles), 0)

    @patch('sys.stdout', new_callable=io.StringIO)
    def test_print_dependency_tree(self, mock_stdout):
        print_dependency_tree(self.graph, 'core-admin')
        expected_output = (
            "core-admin\n"
            "├── core-php\n"
            "└── core-tenant\n"
            "    └── core-php\n"
        )
        self.assertEqual(mock_stdout.getvalue().strip(), expected_output.strip())

    @patch('sys.stdout', new_callable=io.StringIO)
    def test_print_dependency_tree_no_deps(self, mock_stdout):
        print_dependency_tree(self.graph, 'core-php')
        expected_output = "core-php\n"
        self.assertEqual(mock_stdout.getvalue().strip(), expected_output.strip())


    @patch('sys.stdout', new_callable=io.StringIO)
    def test_print_reverse_dependencies(self, mock_stdout):
        print_reverse_dependencies(self.graph, 'core-php')
        expected_output = (
            "├── core-admin\n"
            "├── core-analytics\n"
            "├── core-api\n"
            "└── core-tenant"
        )
        self.assertEqual(mock_stdout.getvalue().strip(), expected_output.strip())

    @patch('sys.stdout', new_callable=io.StringIO)
    def test_print_reverse_dependencies_no_deps(self, mock_stdout):
        print_reverse_dependencies(self.graph, 'core-admin')
        expected_output = "(no modules depend on core-admin)"
        self.assertEqual(mock_stdout.getvalue().strip(), expected_output.strip())

    @patch('deps.find_repos_yaml', return_value='dummy_path.yaml')
    @patch('sys.stdout', new_callable=io.StringIO)
    def test_main_no_args(self, mock_stdout, mock_find_yaml):
        with patch("builtins.open", mock_open(read_data=self.yaml_content)):
            with patch.object(sys, 'argv', ['deps.py']):
                main()
        output = mock_stdout.getvalue()
        self.assertIn("core-admin dependencies:", output)
        self.assertIn("core-tenant dependencies:", output)

    @patch('deps.find_repos_yaml', return_value='dummy_path.yaml')
    @patch('sys.stdout', new_callable=io.StringIO)
    def test_main_module_arg(self, mock_stdout, mock_find_yaml):
        with patch("builtins.open", mock_open(read_data=self.yaml_content)):
            with patch.object(sys, 'argv', ['deps.py', 'core-tenant']):
                main()
        expected_output = (
            "core-tenant dependencies:\n"
            "└── core-php\n"
        )
        self.assertEqual(mock_stdout.getvalue().strip(), expected_output.strip())

    @patch('deps.find_repos_yaml', return_value='dummy_path.yaml')
    @patch('sys.stdout', new_callable=io.StringIO)
    def test_main_reverse_arg(self, mock_stdout, mock_find_yaml):
        with patch("builtins.open", mock_open(read_data=self.yaml_content)):
            with patch.object(sys, 'argv', ['deps.py', '--reverse', 'core-api']):
                main()
        expected_output = (
            "Modules that depend on core-api:\n"
            "└── core-analytics"
        )
        self.assertEqual(mock_stdout.getvalue().strip(), expected_output.strip())

    @patch('deps.find_repos_yaml', return_value='dummy_path.yaml')
    @patch('sys.stdout', new_callable=io.StringIO)
    def test_main_circular_dep(self, mock_stdout, mock_find_yaml):
        with patch("builtins.open", mock_open(read_data=self.circular_yaml_content)):
            with patch.object(sys, 'argv', ['deps.py']):
                with self.assertRaises(SystemExit) as cm:
                    main()
                self.assertEqual(cm.exception.code, 1)
        output = mock_stdout.getvalue()
        self.assertIn("Error: Circular dependencies detected!", output)
        self.assertIn("module-a -> module-b -> module-c -> module-a", output)

    @patch('deps.find_repos_yaml', return_value='dummy_path.yaml')
    @patch('sys.stdout', new_callable=io.StringIO)
    def test_main_non_existent_module(self, mock_stdout, mock_find_yaml):
        with patch("builtins.open", mock_open(read_data=self.yaml_content)):
            with patch.object(sys, 'argv', ['deps.py', 'non-existent-module']):
                with self.assertRaises(SystemExit) as cm:
                    main()
                self.assertEqual(cm.exception.code, 1)
        output = mock_stdout.getvalue()
        self.assertIn("Error: Module 'non-existent-module' not found in repos.yaml.", output)

if __name__ == '__main__':
    unittest.main()
