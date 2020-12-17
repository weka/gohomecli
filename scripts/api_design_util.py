#!/usr/bin/env python3

import sys
import json
import os
from argparse import ArgumentParser

from weka.home import WekaHome


CONFIG_FILE = os.path.expanduser('~/.config/home-cli/config.toml')


def main():
    parser = ArgumentParser()
    parser.add_argument('url')
    parser.add_argument('--original', action='store_true', help='Do not clear strings')
    args = parser.parse_args()
    return run(args)


def run(args):
    with open(CONFIG_FILE) as f:
        for line in f.readlines():
            if 'api_key = ' in line:
                api_key = line.strip().split(' = ', 1)[1].strip().strip('"')
                break
        else:
            raise RuntimeError('API key not found in %s' % CONFIG_FILE)
    wh = WekaHome('api.fries.home.weka.io', api_key)
    result = wh.get('api/v3/%s' % args.url)
    if not args.original:
        result = clear_strings(result)
    print(json.dumps(result, indent=4))


def clear_strings(obj):
    if not isinstance(obj, dict):
        return obj
    for key, value in list(obj.items()):
        if isinstance(value, str):
            obj[key] = 'STRING'
        elif isinstance(value, dict):
            clear_strings(value)
    return obj


if __name__ == '__main__':
    sys.exit(main())
