import requests

def compare(expected, actual, url="http://localhost:8080/compare"):
    payload = {
        "actual": actual,
        "expected": expected,
    }
    r = requests.get(url, json=payload)
    return r.status_code, r.text
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