#!/usr/bin/env python
# coding: utf-8
import sys
import socket
import time
import os
import random

import logging

log = logging.getLogger(__name__)


class GrimReaper(object):
    def __init__(self, socket_path='/tmp/grimreaper.socket'):
        self.path = socket_path
        self.socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self._connect()

    def _is_connected(self):
        try:
            self.socket.recv(0)
        except socket.error as e:
            return self._connect()
        else:
            return True

    def _connect(self):
        try:
            self.socket.connect(self.path)
        except socket.error as e:
            log.error('Cannot connect to socket "%s": %s', self.path, e)
            self.on_connection_error(e)

            if e.errno == socket.errno.ENOENT:
                # No such file or directory
                return False
            elif e.errno == socket.errno.ECONNREFUSED:
                # Connection refused
                return False

            raise
        return True

    def register(self, timeout=None, pid=None):
        if timeout is None:
            timeout = 30

        if pid is None:
            pid = os.getpid()

        if self._is_connected():
            self.socket.sendall('start:%s:%s' % (pid, timeout))
        else:
            print("cannot send start")

    def unregister(self, pid=None):
        if pid is None:
            pid = os.getpid()

        if self._is_connected():
            self.socket.sendall('done:%s' % pid)
        else:
            print("cannot send done")

    def on_connection_error(self, exc):
        pass
        print(exc)


    # TODO: context manager, decorator


def open_close(reaper):
    timeout = random.randint(1, 15)

    reaper.register(timeout=timeout)

    delay = random.randint(1, 10)
    print("Will sleep for %s seconds." % delay)
    time.sleep(delay)

    reaper.unregister()


def main(args):
    reaper = GrimReaper()

    if len(args) == 2 and args[1] == 'open-close':
        open_close(reaper)
    else:
        return


if __name__ == '__main__':
    main(sys.argv)
