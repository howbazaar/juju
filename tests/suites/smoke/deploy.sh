run_deploy() {
    echo

    file="${TEST_DIR}/smoke_deploy.txt"

    bootstrap lxd "smoke_test_deploy" "${file}"
    CHK=$(cat "${file}" | grep -i "ERROR" || true)
    if [ -n "${CHK}" ]; then
        printf "\\nFound some issues"
        cat "${file}" | xargs echo -I % "\\n%"
        exit 1
    fi
}

test_deploy() {
    if [ -n "${SKIP_SMOKE_DEPLOY:-}" ]; then
        echo "==> SKIP: Asked to skip smoke deploy tests"
        return
    fi

    (
        set -e

        # Check that deploy runs on LXD
        run "deploy"
    )
}
