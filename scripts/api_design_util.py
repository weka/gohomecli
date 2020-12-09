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
    args = parser.parse_args()
    return run(args)


def run(args):
    with open(CONFIG_FILE) as f:
        for line in f.readlines():
            if 'api_key = ' in line:
                api_key = line.strip().split(' = ', 1)[1]
                break
        else:
            raise RuntimeError('API key not found in %s' % CONFIG_FILE)
    wh = WekaHome('api.fries.home.weka.io', api_key)
    print(json.dumps(clear_strings(wh.get('api/v3/%s' % args.url)), indent=4))


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
