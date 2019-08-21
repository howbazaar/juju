test_controller() {
    if [ -n "${SKIP_CONTROLLER:-}" ]; then
        echo "==> SKIP: Asked to skip controller tests"
        return
    fi

    test_mongo_memory_profile
}
