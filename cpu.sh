#!/bin/sh

TARGET_CPU=60
DURATION=${1:-60}
CPU_CORES=$(nproc)
TASKS=$(( (TARGET_CPU * CPU_CORES) / 100 ))

echo "Starting $TASKS tasks to load CPU to ~$TARGET_CPU% for $DURATION seconds..."

i=1
while [ "$i" -le "$TASKS" ]; do
    (
        end=$(( $(date +%s) + DURATION ))  # 使用 `date +%s` 获取当前时间戳
        while [ $(date +%s) -lt "$end" ]; do
            md5sum /dev/zero >/dev/null 2>&1
        done
    ) &
    i=$((i + 1))
done

wait
echo "CPU load test finished."