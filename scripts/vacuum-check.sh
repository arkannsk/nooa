#!/usr/bin/env bash
set -uo pipefail

TMP_DIR="$(mktemp -d)"
FAIL=0
PORT=9090

# Массивы для сбора итогов
declare -a RESULT_NAMES
declare -a RESULT_SCORES
declare -a RESULT_WARNINGS
declare -a RESULT_INFOS
declare -a RESULT_ERRORS
declare -a RESULT_GRADES

# Убивает все процессы, слушающие указанный порт
kill_port() {
    local pids
    pids=$(lsof -ti ":$PORT" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill -9 2>/dev/null || true
        sleep 0.5
    fi
}

# Убивает процесс и всю его группу (включая дочерние процессы)
kill_process_group() {
    local pid="$1"
    if [ -n "$pid" ] && [ "$pid" -gt 0 ] 2>/dev/null; then
        kill -- "-$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    fi
}

# Парсит вывод vacuum и возвращает: score warnings infos grade
parse_vacuum_output() {
    local output="$1"

    local score="?"
    local warnings="?"
    local infos="?"
    local grade="?"

    # Quality Score: 99/100 [A+]  — [^\]]+ чтобы поймать A+, F и т.д.
    score=$(echo "$output" | grep -oP 'Quality Score:\s*\K\d+' || echo "?")
    grade=$(echo "$output" | grep -oP 'Quality Score:\s*\d+/100\s*\[\K[^\]]+' || echo "?")

    # Берём первую таблицу (категории), ищем строку total.
    # vacuum выводит две таблицы с total-ом; нам нужна первая — категории.
    # Первая таблица идёт после разделителя ────────────────  ──────────── ...
    local first_total
    first_total=$(echo "$output" | sed -n '/^\s*category\s*/,/total/p' | grep 'total' | head -1)
    if [ -n "$first_total" ]; then
        warnings=$(echo "$first_total" | awk '{print $3}')
        infos=$(echo "$first_total" | awk '{print $4}')
    fi

    echo "$score $warnings $infos $grade"
}

trap 'rm -rf "$TMP_DIR"; kill_port' EXIT

# Убедимся что порт свободен перед началом
kill_port

EXAMPLES=(
    "examples/elval-integration/01_basic_types"
    "examples/elval-integration/02_files_stream"
    "examples/elval-integration/03_nested"
    "examples/elval-integration/04_slice_maps"
    "examples/elval-integration/05_generics"
    "examples/elval-integration/06_polymorphism"
    "examples/elval-integration/07_rewrite"
    "examples/elval-integration/08_http_params"
)

for EXAMPLE in "${EXAMPLES[@]}"; do
    NAME=$(basename "$EXAMPLE")
    echo ""
    echo "========================================"
    echo "  Validating: $EXAMPLE (port $PORT)"
    echo "========================================"

    # Убеждаемся что порт свободен перед запуском
    kill_port

    set -m  # Включает job control для создания новой процесс-группы
    go run "$EXAMPLE/main.go" &
    SERVER_PID=$!
    set +m
    echo "  Server PID: $SERVER_PID"

    # Ждём пока сервер поднимется (до 30 сек)
    READY=0
    for i in $(seq 1 60); do
        if curl -sf "http://localhost:$PORT/openapi.json" -o /dev/null 2>&1; then
            READY=1
            break
        fi
        sleep 0.5
    done

    if [ "$READY" -eq 0 ]; then
        echo "  FAILED: Server did not start in time"
        kill_process_group "$SERVER_PID"
        FAIL=1

        RESULT_NAMES+=("$NAME")
        RESULT_SCORES+=("-")
        RESULT_WARNINGS+=("-")
        RESULT_INFOS+=("-")
        RESULT_ERRORS+=("-")
        RESULT_GRADES+=("FAIL")
        continue
    fi

    SPEC="$TMP_DIR/${NAME}.json"
    curl -sf "http://localhost:$PORT/openapi.json" -o "$SPEC" || {
        echo "  FAILED: Could not download spec"
        kill_process_group "$SERVER_PID"
        FAIL=1

        RESULT_NAMES+=("$NAME")
        RESULT_SCORES+=("-")
        RESULT_WARNINGS+=("-")
        RESULT_INFOS+=("-")
        RESULT_ERRORS+=("-")
        RESULT_GRADES+=("FAIL")
        continue
    }

    # Проверяем что спецификация не пустая
    if [ ! -s "$SPEC" ]; then
        echo "  FAILED: Spec file is empty"
        kill_process_group "$SERVER_PID"
        FAIL=1

        RESULT_NAMES+=("$NAME")
        RESULT_SCORES+=("-")
        RESULT_WARNINGS+=("-")
        RESULT_INFOS+=("-")
        RESULT_ERRORS+=("-")
        RESULT_GRADES+=("FAIL")
        continue
    fi

    # Убиваем сервер (всю процесс-группу)
    kill_process_group "$SERVER_PID"

    echo "  Spec saved: $SPEC"

    # Проверяем vacuum
    if command -v vacuum > /dev/null 2>&1; then
        VACUUM_OUTPUT=$(vacuum lint "$SPEC" --details --no-banner --no-style 2>&1 || true)
        echo "$VACUUM_OUTPUT"

        read -r SCORE WARNINGS INFOS GRADE <<< "$(parse_vacuum_output "$VACUUM_OUTPUT")"

        RESULT_NAMES+=("$NAME")
        RESULT_SCORES+=("$SCORE")
        RESULT_WARNINGS+=("$WARNINGS")
        RESULT_INFOS+=("$INFOS")
        RESULT_ERRORS+=("0")
        RESULT_GRADES+=("$GRADE")
    else
        echo "  WARNING: vacuum not found, skipping validation"

        RESULT_NAMES+=("$NAME")
        RESULT_SCORES+=("N/A")
        RESULT_WARNINGS+=("N/A")
        RESULT_INFOS+=("N/A")
        RESULT_ERRORS+=("N/A")
        RESULT_GRADES+=("N/A")
    fi
done

# ==================== Итоговая сводка ====================
echo ""
echo "============================================================"
echo "  Vacuum Validation Summary"
echo "============================================================"
printf "%-22s %8s %10s %8s %6s\n" "Example" "Score" "Warnings" "Info" "Grade"
printf "%-22s %8s %10s %8s %6s\n" "----------------------" "--------" "----------" "--------" "------"

for i in "${!RESULT_NAMES[@]}"; do
    printf "%-22s %8s %10s %8s %6s\n" \
        "${RESULT_NAMES[$i]}" \
        "${RESULT_SCORES[$i]}/100" \
        "${RESULT_WARNINGS[$i]}" \
        "${RESULT_INFOS[$i]}" \
        "[${RESULT_GRADES[$i]}]"
done

echo "============================================================"

if [ "$FAIL" -ne 0 ]; then
    echo "Some examples failed to start."
    exit 1
fi

echo "Vacuum validation complete."
