#!/usr/bin/env python3
"""
Tool for pulling dependent repositories and performing other operations when building PMM
"""
import argparse
import configparser
import logging
import os
import sys

from subprocess import check_output
from pathlib import Path

import yaml
import git

logging.basicConfig(stream=sys.stdout, format='[%(levelname)s] %(asctime)s: %(message)s', level=logging.INFO)

YAML_EXPORT_CONFIG = 'gitmodules.yml'
YAML_CONFIG_OVERRIDE = 'ci.yml'
SUBMODULES_CONFIG = '.gitmodules'

class Builder():
    rootdir = check_output(['git', 'rev-parse', '--show-toplevel']).decode('utf-8').strip()

    def __init__(self):
        self.config_source = SUBMODULES_CONFIG

        self.config_override = self.read_config_override()
        self.config = self.read_config()

        self.merge_configs()
        self.validate_config()

    def read_config_override(self):
        with open(YAML_CONFIG_OVERRIDE, 'r') as f:
            return yaml.load(f, Loader=yaml.FullLoader)

    def read_config(self):
        config = configparser.ConfigParser()
        config.read(self.config_source)

        submodules = []
        for s in config.sections():
            submodules_name = s.split('"')[1]
            submodules_info = dict(config.items(s))
            submodules_info['name'] = submodules_name

            submodules.append(submodules_info)
        return {'deps': submodules}

    def merge_configs(self):
        if self.config_override is not None:
            for override_dep in self.config_override['deps']:

                for dep in self.config['deps']:
                    if dep['name'] == override_dep['name']:
                        if 'url' in override_dep and override_dep['url'] != dep['url']:
                            dep['repo_url_changed'] = True
                        for (k, v) in override_dep.items():
                            dep[k] = v
                        break
                else:
                    logging.error(
                        f'Can"t find {override_dep["name"]} repo from ci.yml in the list of repos from .gitmodules')
                    sys.exit(1)

    # To test the merge, run `python ./ci.py --convert`
    def export_gitmodules_to_yaml(self, target=YAML_EXPORT_CONFIG):
        yaml_config = Path(target)
        if yaml_config.is_file():
            logging.warning('File {} already exists!'.format(target))
            sys.exit(1)
        with open(target, 'w') as f:
            yaml.dump(self.config, f, sort_keys=False)
        sys.exit(0)          

    def validate_config(self):
        for dep in self.config['deps']:
            if not os.path.abspath(dep['path']).startswith(os.getcwd()):
                logging.error(f'For dependency [{dep["name"]} -> {os.path.abspath(dep["path"])}] '
                              f'the path must be located within the working directory [{os.getcwd()}]')
                sys.exit(1)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--convert', help='convert .gitmodules config to yml and merge with ci.yml', action='store_true')

    args = parser.parse_args()

    builder = Builder()

    if args.convert:
        builder.export_gitmodules_to_yaml()
        sys.exit(0)

main()
