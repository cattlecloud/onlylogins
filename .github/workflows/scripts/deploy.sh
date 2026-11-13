#!/usr/bin/env bash

set -euo pipefail

binary="onlylogins"

release=$(curl -s -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/cattlecloud/${binary}/releases/latest)

url=$(echo "${release}" | jq -r '.assets[] | select(.name == "'"${binary}"'") | .url')
dest="/tmp/${binary}"

echo "downloading ${url} into ${dest}"
curl -L \
  -H "Accept: application/octet-stream" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -o "${dest}" -s "${url}"
echo "download complete."

chmod +x "${dest}"

scp "${dest}" "${HOST}:~/${binary}"
ssh -t "${HOST}" "sudo mv ~/${binary} /opt/bin/${binary}"
ssh -t "${HOST}" "sudo systemctl restart ${binary}"
ssh -t "${HOST}" "sudo systemctl status  ${binary}"

rm "${dest}"
echo "done."
