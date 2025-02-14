#!/usr/bin/env python3
"""
Tool for pulling dependent repositories and performing other operations when building PMM
"""
import argparse
import configparser
import logging
import os
import sys

from subprocess import check_output, check_call, call, CalledProcessError
from pathlib import Path
from github import Github, UnknownObjectException

import yaml
import git

logging.basicConfig(stream=sys.stdout, format='[%(levelname)s] %(asctime)s: %(message)s', level=logging.INFO)

YAML_EXPORT_CONFIG = 'gitmodules.yml'
YAML_CONFIG_OVERRIDE = 'ci.yml'
JSON_EXPORT_CONFIG = '.json'
SUBMODULES_CONFIG = '.gitmodules'
GIT_SOURCES_FILE = '.git-sources'
FORK_OWNER = os.environ.get('FORK_OWNER', '')
GITHUB_TOKEN = os.environ.get('GITHUB_API_TOKEN', '')
# example CHANGE_URL : https://github.com/Percona-Lab/pmm-submodules/pull/2167
PR_URL = os.environ.get('CHANGE_URL', '')


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

    def write_custom_config(self, config):
        with open(YAML_CONFIG_OVERRIDE, 'w') as f:
            yaml.dump(config, f, sort_keys=False)

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
                        f'Can"t find {override_dep["name"]} repo from ci.yml in the list of repos in ci-default.yml')
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

    def get_global_branches(self, target_branch_name):
        # this is a key/value structure, where key is a name of the repo and value is a link to the repo
        found_branches = {}
        github_api = Github(GITHUB_TOKEN)
        for dep in self.config['deps']:
            url_parts = dep['url'].split('/')[-2:]
            owners = [url_parts[0]]
            if FORK_OWNER != '':
                owners.append(FORK_OWNER)

            for owner in owners:
                url_parts[0] = owner
                repo_path = '/'.join(url_parts).replace('.git', '')
                try:
                    repo = github_api.get_repo(repo_path)
                except UnknownObjectException:
                    continue

                for branch in repo.get_branches():
                    if target_branch_name == branch.name:
                        logging.info(f'Found branch {target_branch_name} in {dep["name"]} submodule')
                        found_branches[dep['name']] = repo.html_url

        return found_branches

    def create_yaml_config(self, branch_name=''):
        if branch_name == '':
            logging.error('Please provide a branch name to search for')
            sys.exit(1)

        found_branches = self.get_global_branches(branch_name)

        if self.config_override is None:
            self.config_override = {'deps': []}

        # update the override config
        for dep in self.config_override['deps']:
            if dep['name'] in found_branches:
                dep['branch'] = branch_name
                dep['url'] = found_branches[dep['name']]
                found_branches.pop(dep['name'])

        for dep_name, url in found_branches.items():
            self.config_override['deps'].append({'name': dep_name, 'branch': branch_name, 'url': url})

        self.write_custom_config(self.config_override)

    def create_fb_branch(self, branch_name, global_repo=False):
        repo = git.Repo('.')

        git_cmd = repo.git
        for ref in repo.references:
            if branch_name == ref.name:
                git_cmd.checkout(branch_name)
                break
        else:
            git_cmd.checkout('HEAD', b=branch_name)

        found_branches = {}
        if global_repo:
            found_branches = self.get_global_branches(branch_name)

        if self.config_override is None:
            self.config_override = {'deps': []}

        # change old records
        for dep in self.config_override['deps']:
            if dep['name'] in found_branches:
                dep['branch'] = branch_name
                dep['url'] = found_branches[dep['name']]
                found_branches.pop(dep['name'])

        for dep_name, url in found_branches.items():
            self.config_override['deps'].append({'name': dep_name, 'branch': branch_name, 'url': url})

        self.write_custom_config(self.config_override)
        repo.git.add(['ci.yml', ])
        repo.index.commit(f'Create feature build: {branch_name}')
        origin = repo.remote(name='origin')
        try:
            origin.push()
        except git.exc.GitCommandError:  # Could be due to no upstream branch.
            logging.warning(f'Failed to push {branch_name}. This could be due to no matching upstream branch.')
            logging.info(
                f'Reattempting to push {branch_name} using a lower-level command which also sets upstream branch.')
            push_output = repo.git.push('--set-upstream', 'origin', branch_name)
            logging.info(f'Push output was: {push_output}')

        logging.info('Last ci.yml was pushed')

        if GITHUB_TOKEN:
            github_api = Github(GITHUB_TOKEN)
            repo = github_api.get_repo('Percona-Lab/pmm-submodules')
            pr = repo.get_pulls(base='v3', head=f'Percona-Lab:{branch_name}')
            # TODO we can use totalCount here: https://github.com/PyGithub/PyGithub/blob/babcbcd04fd5605634855f621b8558afc5cbc515/github/PaginatedList.py#L102
            # but it works pretty strange. It returned count ALL PR from repo without filters
            hasPR = False
            for i in pr:
                hasPR = True
                break
            if not hasPR:
                body = 'Custom branches: \n'
                for dep in self.config_override['deps']:
                    # TODO we need to have link to PR here
                    body = body + dep['name'] + '\n'
                pr = repo.create_pull(
                    title=f'{branch_name} (FB)',
                    body=body,
                    head=branch_name,
                    base='v3',
                    draft=True
                )
                logging.info(
                    f'Pull Request was created: https://github.com/Percona-Lab/pmm-submodules/pull/{pr.number}')
            else:
                logging.info(
                    f'Pull request already exists: https://github.com/Percona-Lab/pmm-submodules/pull/{pr[0].number}')
        else:
            logging.info('Branch was created')
            logging.info(
                f'Need to create PR now: https://github.com/Percona-Lab/pmm-submodules/compare/{branch_name}?expand=1')

    def get_deps(self):
        with open(GIT_SOURCES_FILE, 'w+') as f:
            for dep in self.config['deps']:
                path = os.path.join(self.rootdir, dep['path'])

                def repo_cloned():
                    return os.path.exists(os.path.join(self.rootdir, path))

                if dep.get('repo_url_changed') and repo_cloned():
                    check_call(f'rm -rf {path}'.split())

                if not repo_cloned():
                    target_branch = dep['branch']
                    target_url = dep['url']
                    check_call(
                        f'git clone --depth 1 --single-branch --branch {target_branch} {target_url} {path}'.split())
                else:
                    logging.info(f'Files in the path for {dep["name"]} is already exist')
                call(['git', 'pull', '--ff-only'], cwd=path)
                commit_id = switch_branch(path, dep['branch'])

                dep_name_underscore = dep['name'].replace('-', '_')
                f.write(f'export {dep_name_underscore}_commit={commit_id}')
                f.write(f'export {dep_name_underscore}_branch={dep["branch"]}\n')
                f.write(f'export {dep_name_underscore}_url={dep["url"]}\n')

    def check_deps(self):
        outdated_branches_message = 'Looks like there are outdated source branches.\n Please update them and restart ' \
                                    'the job'
        outdated_branches = []
        submodules_url = '/'.join(PR_URL.split('/')[3:-2])
        pull_number = PR_URL.split('/')[-1:][0]
        GH_ACTIONS_TOKEN = GITHUB_TOKEN

        if GH_ACTIONS_TOKEN == '':
            logging.warning('there is no GITHUB_TOKEN, looking for GH_API_TOKEN')
            GH_ACTIONS_TOKEN = os.environ.get('GH_API_TOKEN', '')

        github_api = Github(GH_ACTIONS_TOKEN)

        # it's not a good idea to use config_override here. Maybe we can add 'custom' key?
        for dep in self.config_override['deps']:
            if 'url' in dep:
                target_url = dep['url']
            else:
                target_url = next(item for item in self.config['deps'] if item["name"] == dep['name'])['url']
            repo_path = '/'.join(target_url.split('/')[-2:])
            target_branch = dep['branch']
            repo = github_api.get_repo(repo_path)
            org = repo.organization.name if repo.organization else repo.owner.login
            head = f'{org}:{target_branch}'
            pulls_list = repo.get_pulls('open', 'updated', 'asc', 'main', head)
            if not pulls_list.totalCount:
                continue

            pull = repo.get_pull(pulls_list[0].number)
            if pull.mergeable_state in ['behind', 'dirty']:
                outdated_branches.append(pull.html_url)

        if outdated_branches:
            for branch_url in outdated_branches:
                outdated_branches_message += f'\n {branch_url}'

            repo = github_api.get_repo(submodules_url)
            pull = repo.get_pull(int(pull_number))
            pull.create_issue_comment(outdated_branches_message)
            sys.exit(1)

    def validate_config(self):
        for dep in self.config['deps']:
            if not os.path.abspath(dep['path']).startswith(os.getcwd()):
                logging.error(f'For dependency [{dep["name"]} -> {os.path.abspath(dep["path"])}] '
                              f'the path must be located within the working directory [{os.getcwd()}]')
                sys.exit(1)


def switch_branch(path, branch):
    # 'symbolic-ref' works only if we are on a branch. If we use a commit, we run 'rev-parse' instead.
    try:
        cur_branch = check_output('git symbolic-ref --short HEAD'.split(), cwd=path).decode().strip()
    except CalledProcessError:
        cur_branch = check_output('git rev-parse HEAD'.split(), cwd=path).decode().strip()
    if cur_branch != branch:
        branches = check_output('git ls-remote --heads origin'.split(), cwd=path)
        branches = [line.split("/")[-1] for line in branches.decode().strip().split("\n")]

        if branch in branches:
            print(f'Switch to branch: {branch} (from {cur_branch})')
            check_call(f'git remote set-branches origin {branch}'.split(), cwd=path)
            check_call(f'git fetch --depth 1 origin {branch}'.split(), cwd=path)
            check_call(f'git checkout {branch}'.split(), cwd=path)
        else:
            logging.error(f'Can\' find branch: {branch} in {path}')
            sys.exit(1)

    return check_output('git rev-parse HEAD'.split(), cwd=path).decode("utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--prepare', help='prepare feature build')
    parser.add_argument('--global', '-g', dest='global_repo', help='find and use all branches with this name',
                        action='store_true')
    parser.add_argument('--convert', help='convert .gitmodules config to yml and merge with ci.yml', action='store_true')
    parser.add_argument('--create-config', '-c', dest='create_config', help='find all branches with the same name and create the ci.yml config')

    args = parser.parse_args()

    builder = Builder()

    if args.convert:
        builder.export_gitmodules_to_yaml()
        sys.exit(0)

    if args.prepare:
        builder.create_fb_branch(args.prepare, args.global_repo)
        sys.exit(0)
    
    if args.create_config:
        builder.create_yaml_config(args.create_config)
        sys.exit(0)

    builder.check_deps()
    builder.get_deps()


main()
