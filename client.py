#!/usr/bin/env python
# coding: utf-8
import sys
import socket
import time
import os
import random


def spam(sock, messages):
    for message in messages:
        print("Sending %s" % message)
        sock.sendall(message)
        time.sleep(1)


def open_close(sock):
    timeout = random.randint(1, 15)
    sock.sendall('start:%s:%s' % (os.getpid(), timeout))
    delay = random.randint(1, 10)
    print("Will sleep for %s seconds." % delay)
    time.sleep(delay)
    sock.sendall('done:%s' % os.getpid())


def main(args):
    sock_path = "/tmp/grimreaper.socket"
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    sock.connect(sock_path)

    if len(args) == 2 and args[1] == 'open-close':
        open_close(sock)
    else:
        spam(sock, args[1:])


if __name__ == '__main__':
    main(sys.argv)
