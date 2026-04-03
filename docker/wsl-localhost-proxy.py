#!/usr/bin/env python3
import argparse
import socket
import subprocess
import sys
import threading


BUFFER_SIZE = 65536

WSL_BRIDGE_CODE = r"""
import os
import socket
import sys
import threading

s = socket.create_connection(("127.0.0.1", int(sys.argv[1])))

def socket_to_stdout():
    try:
        while True:
            data = s.recv(65536)
            if not data:
                break
            os.write(1, data)
    except Exception:
        pass
    finally:
        try:
            s.shutdown(socket.SHUT_RDWR)
        except Exception:
            pass

thread = threading.Thread(target=socket_to_stdout, daemon=True)
thread.start()

try:
    while True:
        data = os.read(0, 65536)
        if not data:
            break
        s.sendall(data)
except Exception:
    pass
finally:
    try:
        s.shutdown(socket.SHUT_WR)
    except Exception:
        pass

thread.join()
s.close()
"""


def pipe_client_to_process(client, process):
    try:
        while True:
            data = client.recv(BUFFER_SIZE)
            if not data:
                break
            process.stdin.write(data)
            process.stdin.flush()
    except Exception:
        pass
    finally:
        try:
            process.stdin.close()
        except Exception:
            pass


def pipe_process_to_client(client, process):
    read = process.stdout.read1 if hasattr(process.stdout, "read1") else process.stdout.read

    try:
        while True:
            data = read(BUFFER_SIZE)
            if not data:
                break
            client.sendall(data)
    except Exception:
        pass
    finally:
        try:
            client.shutdown(socket.SHUT_RDWR)
        except Exception:
            pass
        client.close()


def handle_client(client, distro, target_port):
    process = subprocess.Popen(
        ["wsl.exe", "-d", distro, "--", "python3", "-u", "-c", WSL_BRIDGE_CODE, str(target_port)],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        bufsize=0,
    )

    threads = [
        threading.Thread(target=pipe_client_to_process, args=(client, process), daemon=True),
        threading.Thread(target=pipe_process_to_client, args=(client, process), daemon=True),
    ]

    for thread in threads:
        thread.start()

    for thread in threads:
        thread.join()

    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        process.kill()


def serve(listen_host, listen_port, distro, target_port):
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind((listen_host, listen_port))
    server.listen()

    print(f"Proxy listening on http://{listen_host}:{listen_port} -> WSL {distro}:127.0.0.1:{target_port}")
    sys.stdout.flush()

    while True:
        client, _ = server.accept()
        thread = threading.Thread(
            target=handle_client,
            args=(client, distro, target_port),
            daemon=True,
        )
        thread.start()


def main():
    parser = argparse.ArgumentParser(description="Forward a Windows localhost TCP port into a WSL distro.")
    parser.add_argument("--listen-host", default="127.0.0.1")
    parser.add_argument("--listen-port", type=int, required=True)
    parser.add_argument("--target-port", type=int, required=True)
    parser.add_argument("--distro", default="Debian")
    args = parser.parse_args()

    serve(args.listen_host, args.listen_port, args.distro, args.target_port)


if __name__ == "__main__":
    main()
