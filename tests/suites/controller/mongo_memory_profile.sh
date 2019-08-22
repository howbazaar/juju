cat_mongo_service() {
    echo $(juju run -m controller --machine 0 'cat /lib/systemd/system/juju-db/juju-db.service' | grep "^ExecStart")
}

run_mongo_memory_profile() {
    echo

    file="${TEST_DIR}/mongo_memory_profile.txt"

    ensure_controller "mongo-memory-profile" "${file}"

    check_contains "$(cat_mongo_service)" wiredTigerCacheSizeGB

    juju controller-config mongo-memory-profile=low

    sleep 5

    check_not_contains "$(cat_mongo_service)" wiredTigerCacheSizeGB

    # Set the value back in case we are reusing a controller
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
