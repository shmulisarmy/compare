import json
import subprocess
import shutil
import requests
from typing import Any, Tuple, Optional


def to_terminal_safe_json_string(v: Any) -> str:
    return json.dumps(v)


def compare(expected: Any, actual: Any, url: str = "") -> Tuple[int, str, Optional[Exception]]:
    compare_path = shutil.which("compare")
    if compare_path:
        # Use local compare binary
        terminal_command = f"{compare_path} -expected='{to_terminal_safe_json_string(expected)}' -actual='{to_terminal_safe_json_string(actual)}'"
        try:
            result = subprocess.run(
                terminal_command, shell=True, check=True, capture_output=True, text=True
            )
            return 0, result.stdout, None
        except subprocess.CalledProcessError as e:
            return 1, e.stdout + e.stderr, e
    else:
        print("compare is not a command")

    if not url:
        url = "http://localhost:8080/compare"

    payload = {"expected": expected, "actual": actual}

    try:
        response = requests.get(url, json=payload)
        return response.status_code, response.text, None
    except Exception as e:
        return 0, "", e




print(compare({
    "name": "bob",
    "age": 30,
    "friends": [
        {
            "name": "charlie"
        },
        "bob",
        "gil",
    ]
}, {
    "name": "alice",
    "age": 30,
    "friends": [
        {
            "name": "charlie2"
        },
        "bob",
        "gil",
    ]
})[1])