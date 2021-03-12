#! /usr/bin/env python3

"""It's an "instance file" used for tests.
By convention, all instance file have a ".lua" extension (which is why the
Python code is inside "test_instance.lua"). And this is a description of why
Python was used instead of lua for the instance file: "For testing, we needed a
simple instance that can customize signal processing. First of all I do it by
using pure lua with luaposix module, but Mons insisted that the luaposix module
is a bad way and shouldn't try to handle signals from pure lua. And I will be
honest, the user may have problems installing this module. To do a signal
processing on the tarantool by overriding the standard processing is a bad idea
in my opinion. Therefore, it was decided to use python for these purposes.

All hacks are done by professionals please don't try to repeat it yourself!
"""

import os
import signal
import sys
import time


def handler(signum, frame):
    """Signal handler"""
    ignore = os.getenv('INSTSIGIGNORE')
    if not(ignore and ignore.lower() == 'true'):
        sys.exit(0)


def main():
    # Set a signal handler
    signal.signal(signal.SIGTERM, handler)
    signal.signal(signal.SIGINT, handler)

    while True:
        time.sleep(1)


main()
