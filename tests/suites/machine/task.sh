test_machine() {
    if [ "$(skip 'test_machine')" ]; then
        echo "==> TEST SKIPPED: test_machine"
        return
    fi

    echo "==> Checking for dependencies"
    check_dependencies juju

    file="${TEST_DIR}/test_machine.txt"

    bootstrap "test_machine" "${file}"

    # Test that need to be run are added here!
    test_logs

    destroy_controller "test_machine"
}
