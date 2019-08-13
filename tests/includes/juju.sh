bootstrap() {
    local provider name

    provider=${1}
    shift

    name=${1}
    shift

    output=${1}
    shift

    echo "====> Bootstrapping juju"
    if [ -n "${output}" ]; then
        juju bootstrap "${provider}" "${name}" "$@" > "${output}" 2>&1
    else
        juju bootstrap "${provider}" "${name}" "$@"
    fi
    echo "${name}" >> "${TEST_DIR}/jujus"

    echo "====> Bootstrapped juju"
}

destroy() {
    local name

    name=${1}
    shift

    file="${TEST_DIR}/${name}_destroy.txt"

    echo "====> Destroying juju ${name}"
    echo "${name}" | xargs -I % juju destroy-controller --destroy-all-models -y % > "${file}" 2>&1
    CHK=$(cat "${file}" | grep -i "ERROR" || true)
    if [ -n "${CHK}" ]; then
        printf "\\nFound some issues"
        cat "${file}" | xargs echo -I % "\\n%"
        exit 1
    fi
    echo "====> Destroyed juju ${name}"
}

cleanup_jujus() {
    echo "====> Cleaning up jujus"

    while read -r juju_name; do
        destroy "${juju_name}"
    done < "${TEST_DIR}/jujus"
}
