<dl>
  <h1>
    <div align=center>GitOops!</div>
    <div align=center>😱</div>
  </h1>
  <p align="center"><i>all paths lead to clouds</i></p>
  <br />
</dl>

GitOops is a tool to help attackers and defenders identify lateral movement and privilege escalation paths in GitHub organizations by abusing CI/CD pipelines and GitHub access controls.

It maps relationships between your GitHub organization and environment variables in your CI/CD systems. It uses any Bolt-compatible graph database, so you can query your attack paths with openCypher:

```
MATCH p=(:User{login:"alice"})-[*..5]->(v:EnvironmentVariable)
WHERE v.name =~ ".*SECRET.*"
RETURN p
```

<dl>
  <p align="center">
    <img src="./docs/screenshot.png">
  </p>
</dl>

GitOops takes inspiration from tools like [Bloodhound](https://github.com/BloodHoundAD/BloodHound) and [Cartography](https://github.com/lyft/cartography).

Check out the [docs](docs/README.md) and [more example queries](./docs/examples.md).
