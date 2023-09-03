# webtail.exe python foo.py
# python bar.py

import sys
import random
import time
import threading

# List of sample messages
messages = [
    "This is message 1",
    "Message number 2 here",
    "Yet another message, number 3",
    "Message 4",
    "Final message, number 5"
]

# Function to print random messages to stdout
def print_to_stdout():
    while True:
        msg = random.choice(messages)
        sys.stdout.write(f"stdout: {msg}\n")
        sys.stdout.flush()
        time.sleep(random.uniform(0.1, 1))

# Function to print random messages to stderr
def print_to_stderr():
    while True:
        msg = random.choice(messages)
        sys.stderr.write(f"stderr: {msg}\n")
        sys.stderr.flush()
        time.sleep(random.uniform(0.1, 1))

# Create and start threads
stdout_thread = threading.Thread(target=print_to_stdout)
stderr_thread = threading.Thread(target=print_to_stderr)

stdout_thread.start()
stderr_thread.start()

stdout_thread.join()
stderr_thread.join()

