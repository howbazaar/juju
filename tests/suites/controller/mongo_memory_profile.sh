run_mongo_memory_profile() {
    echo

    file="${TEST_DIR}/mongo_memory_profile.txt"

    echo ${file}

    ensure_controller "mongo-memory-profile" "${file}"

    LINE=$(juju run -m controller --machine 0 'cat /lib/systemd/system/juju-db/juju-db.service' | grep "^ExecStart")
    CHK=$(echo "${LINE}" | grep wiredTigerCacheSizeGB || true)
    if [ -n "${CHK}" ]; then
        printf "unexpected wiredTigerCacheSizeGB found in output\n\n%s\n" "${LINE}"
        exit 1
    else
        printf "success: wiredTigerCacheSizeGB not found in output\n"
    fi

    juju controller-config mongo-memory-profile=low

    sleep 5

    LINE=$(juju run -m controller --machine 0 'cat /lib/systemd/system/juju-db/juju-db.service' | grep "^ExecStart")
    CHK=$(echo "${LINE}" | grep wiredTigerCacheSizeGB || true)
    if [ -n "${CHK}" ]; then
        printf "success: wiredTigerCacheSizeGB found as expected\n"
    else
        printf "wiredTigerCacheSizeGB arg missing from output\n\n%s\n" "${LINE}"
        exit 1
    fi

    juju controller-config mongo-memory-profile=default

    sleep 5

    destroy_model "mongo-memory-profile"
}

test_mongo_memory_profile() {
    if [ -n "${SKIP_CONTROLLER_MONGO_MEMORY_PROFILE:-}" ]; then
        echo "==> SKIP: Asked to skip controller mongo memory profile tests"
        return
    fi

    (
        set -e

        cd ../

        run "mongo memory profile"
    )
}
