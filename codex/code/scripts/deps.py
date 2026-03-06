
import os
import sys
import yaml

def find_repos_yaml():
    """Traverse up from the current directory to find repos.yaml."""
    current_dir = os.getcwd()
    while current_dir != '/':
        repos_yaml_path = os.path.join(current_dir, 'repos.yaml')
        if os.path.exists(repos_yaml_path):
            return repos_yaml_path
        current_dir = os.path.dirname(current_dir)
    return None

def parse_dependencies(repos_yaml_path):
    """Parses the repos.yaml file and returns a dependency graph."""
    with open(repos_yaml_path, 'r') as f:
        data = yaml.safe_load(f)

    graph = {}
    repos = data.get('repos', {})
    for repo_name, details in repos.items():
        graph[repo_name] = details.get('depends', []) or []
    return graph

def find_circular_dependencies(graph):
    """Finds circular dependencies in the graph using DFS."""
    visiting = set()
    visited = set()
    cycles = []

    def dfs(node, path):
        visiting.add(node)
        path.append(node)

        for neighbor in graph.get(node, []):
            if neighbor in visiting:
                cycle_start_index = path.index(neighbor)
                cycles.append(path[cycle_start_index:] + [neighbor])
            elif neighbor not in visited:
                dfs(neighbor, path)

        path.pop()
        visiting.remove(node)
        visited.add(node)

    for node in graph:
        if node not in visited:
            dfs(node, [])

    return cycles

def print_dependency_tree(graph, module, prefix=""):
    """Prints the dependency tree for a given module."""
    if module not in graph:
        print(f"Module '{module}' not found.")
        return

    print(f"{prefix}{module}")
    dependencies = graph.get(module, [])
    for i, dep in enumerate(dependencies):
        is_last = i == len(dependencies) - 1
        new_prefix = prefix.replace("├──", "│  ").replace("└──", "   ")
        connector = "└── " if is_last else "├── "
        print_dependency_tree(graph, dep, new_prefix + connector)

def print_reverse_dependencies(graph, module):
    """Prints the modules that depend on a given module."""
    if module not in graph:
        print(f"Module '{module}' not found.")
        return

    reverse_deps = []
    for repo, deps in graph.items():
        if module in deps:
            reverse_deps.append(repo)

    if not reverse_deps:
        print(f"(no modules depend on {module})")
    else:
        for i, dep in enumerate(sorted(reverse_deps)):
            is_last = i == len(reverse_deps) - 1
            print(f"{'└── ' if is_last else '├── '}{dep}")

def main():
    """Main function to handle command-line arguments and execute logic."""
    repos_yaml_path = find_repos_yaml()
    if not repos_yaml_path:
        print("Error: Could not find repos.yaml in the current directory or any parent directory.")
        sys.exit(1)

    try:
        graph = parse_dependencies(repos_yaml_path)
    except Exception as e:
        print(f"Error parsing repos.yaml: {e}")
        sys.exit(1)

    cycles = find_circular_dependencies(graph)
    if cycles:
        print("Error: Circular dependencies detected!")
        for cycle in cycles:
            print(" -> ".join(cycle))
        sys.exit(1)

    args = sys.argv[1:]

    if not args:
        print("Dependency tree for all modules:")
        for module in sorted(graph.keys()):
            print(f"\n{module} dependencies:")
            dependencies = graph.get(module, [])
            if not dependencies:
                print("└── (no dependencies)")
            else:
                for i, dep in enumerate(dependencies):
                    is_last = i == len(dependencies) - 1
                    print_dependency_tree(graph, dep, "└── " if is_last else "├── ")
        return

    reverse = "--reverse" in args
    if reverse:
        args.remove("--reverse")

    if not args:
        print("Usage: /core:deps [--reverse] [module_name]")
        sys.exit(1)

    module_name = args[0]

    if module_name not in graph:
        print(f"Error: Module '{module_name}' not found in repos.yaml.")
        sys.exit(1)

    if reverse:
        print(f"Modules that depend on {module_name}:")
        print_reverse_dependencies(graph, module_name)
    else:
        print(f"{module_name} dependencies:")
        dependencies = graph.get(module_name, [])
        if not dependencies:
            print("└── (no dependencies)")
        else:
            for i, dep in enumerate(dependencies):
                is_last = i == len(dependencies) - 1
                connector = "└── " if is_last else "├── "
                print_dependency_tree(graph, dep, connector)


if __name__ == "__main__":
    main()
