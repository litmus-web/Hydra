from subprocess import Popen, PIPE, STDOUT
import time

rust_times = []
uv_times = []
time.sleep(5)
print("starting")

for _ in range(3):
    cmd = f"wrk -t2 -c256 -d30s http://sandman-test/"
    process = Popen(cmd, shell=True, stdout=PIPE, stdin=PIPE, stderr=STDOUT)
    end = time.time() + 35

    lines = []
    print("============= Sandman =============")
    while time.time() < end:
        data = process.stdout.readline().decode().replace('\n', '')
        if data != '':
            print(data)
            lines.append(data)
        time.sleep(0.1)
    process.kill()
    sec = lines[len(lines) - 2]
    rust_times.append(sec)

    cmd = f"wrk -t2 -c256 -d30s http://uvicorn-test:5050/"
    process = Popen(cmd, shell=True, stdout=PIPE, stdin=PIPE, stderr=STDOUT)
    end = time.time() + 35

    lines = []
    print("============= Uvicorn =============")
    while time.time() < end:
        data = process.stdout.readline().decode().replace('\n', '')
        if data != '':
            print(data)
            lines.append(data)
        time.sleep(0.1)
    process.kill()
    sec = lines[len(lines) - 2]
    uv_times.append(sec)

    """cmd = f"wrk -t2 -c256 -d15s http://94.193.246.143:8000/"
    process = Popen(cmd, shell=True, stdout=PIPE, stdin=PIPE, stderr=STDOUT)
    end = time.time() + 18
    print("============= AioHTTP =============")
    while time.time() < end:
        data = process.stdout.readline().decode().replace('\n', '')
        if data != '':
            print(data)
        time.sleep(0.1)
    process.kill()"""

count = 1
for r, u in zip(rust_times, uv_times):
    print(f"Round {count} |  Sandman: ", r, f" Req/Sec         Round {count} |  Uvicorn: ", u, " Req/Sec")
    count += 1
