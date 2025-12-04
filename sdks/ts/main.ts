async function compare(
  expected: any,
  actual: any,
  url: string = "http://localhost:8080/compare"
): Promise<{ status: number; body: string }> {
  const r = await fetch(url, {
    method: "GET",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ expected, actual })
  });

  return { status: r.status, body: await r.text() };
}

compare(
  {
    name: "bob",
    age: 30,
    friends: [{ name: "charlie" }, "bob", "gil"]
  },
  {
    name: "alice",
    age: 30,
    friends: [{ name: "charlie2" }, "bob", "gil"]
  }
).then(res => console.log(res.body));
