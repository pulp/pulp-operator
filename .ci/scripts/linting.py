#!/usr/bin/env python3
import glob
import os
import shutil
import stat
import sys
import tempfile
import yaml

from ansible.plugins.filter.core import FilterModule
from jinja2 import Environment, FileSystemLoader


DEFAULT_SETTINGS = {
    "ansible_operator_meta": {"namespace": "lint", "name": "test"},
    "deployment_type": "pulp",
}


def lookup(*args, **kwargs):
    return "pulp_lookup"


def main():
    operator_root_dir = os.getcwd()
    crs = glob.glob(f"{operator_root_dir}/config/samples/p*.yaml")
    roles = glob.glob(f"{operator_root_dir}/roles/*")
    for cr in crs:
        with open(cr) as config_file:
            try:
                config_in_file = yaml.safe_load(config_file)
                if not config_in_file.get("spec"):
                    continue
                config = config_in_file["spec"]
                # Add any missing value from the list of defaults
                for key, value in DEFAULT_SETTINGS.items():
                    if key not in config:
                        config[key] = value
                print("\nLoaded vars from " "{path}\n".format(path=cr))
            except yaml.YAMLError as exc:
                print(exc)
                exit()

        for role in roles:
            write_template_section(config, cr, role, operator_root_dir)


def to_nice_yaml(data):
    """Implement a filter for Jinja 2 templates to render human readable YAML."""
    return yaml.dump(data, indent=2, allow_unicode=True, default_flow_style=False)


def write_template_section(config, cr, role_dir, operator_root_dir, verbose=False):
    """
    Template or copy all files for the section.
    """
    env = Environment(
        loader=FileSystemLoader(
            [
                role_dir,  # The scpecified role folder
                "templates",  # The default templates folder
            ]
        )
    )
    ansible_filters = FilterModule().filters()
    env.filters.update(ansible_filters)
    env.globals["lookup"] = lookup
    files_templated = 0
    files_copied = 0
    with open(f"{role_dir}/defaults/main.yml") as default_file:
        default_vars = yaml.safe_load(default_file)
        for key, value in default_vars.items():
            config[key] = value
    if os.path.exists(f"{role_dir}/vars/main.yml"):
        with open(f"{role_dir}/vars/main.yml") as vars_file:
            _vars = yaml.safe_load(vars_file)
            for key, value in _vars.items():
                config[key] = value
    with open(f"{operator_root_dir}/playbooks/pulp.yml") as playbook_file:
        playbook_vars = yaml.safe_load(playbook_file)[0]["vars"]
        for key, value in playbook_vars.items():
            config[key] = value
    config["pulp_combined_settings"] = config["default_settings"]

    for relative_path in generate_relative_path_set(role_dir):
        cr_name = cr.split("/")[-1].replace(".yaml", "")
        role_name = role_dir.split("/")[-1]
        filename = relative_path.split("/")[-1]
        destination_relative_path = f"lint/{cr_name}/{role_name}/{filename}"
        necessary_dir_structure = os.path.dirname(
            os.path.join(operator_root_dir, destination_relative_path)
        )

        if not os.path.exists(necessary_dir_structure):
            os.makedirs(necessary_dir_structure)

        if relative_path.endswith(".j2"):
            env.filters["to_yaml"] = to_nice_yaml
            template = env.get_template(relative_path)
            destination = destination_relative_path[: -len(".j2")]
            write_template_to_file(
                template,
                operator_root_dir,
                destination,
                config,
            )
            files_templated += 1
            if verbose:
                print(f"Templated file: {relative_path}")
        else:
            if destination_relative_path.endswith(".copy"):
                destination_relative_path = destination_relative_path[: -len(".copy")]
            shutil.copyfile(
                os.path.join(role_dir, relative_path),
                os.path.join(operator_root_dir, destination_relative_path),
            )
            files_copied += 1
            if verbose:
                print(f"Copied file: {relative_path}")

    return 0


def generate_relative_path_set(root_dir):
    """
    Create a set of relative paths within the specified directory.
    """
    applicable_paths = set()
    for root, dirs, files in os.walk(root_dir, topdown=False):
        for file_name in files:
            template_abs_path = os.path.join(root, file_name)
            template_relative_path = os.path.relpath(template_abs_path, root_dir)
            if template_relative_path.startswith("template"):
                applicable_paths.add(template_relative_path)
    return applicable_paths


def write_template_to_file(template, operator_root_dir, relative_path, config):
    """
    Render template with values from the config and write it to the target plugin directory.
    """

    with tempfile.NamedTemporaryFile(
        mode="w", dir=operator_root_dir, delete=False
    ) as fd_out:
        tempfile_path = fd_out.name
        fd_out.write(template.render(**config))
        fd_out.write("\n")

        destination_path = os.path.normpath(
            os.path.join(operator_root_dir, relative_path)
        )
        os.rename(tempfile_path, destination_path)

        mode = stat.S_IRUSR | stat.S_IWUSR | stat.S_IRGRP | stat.S_IWGRP | stat.S_IROTH
        os.chmod(destination_path, mode)


if __name__ == "__main__":
    sys.exit(main())
