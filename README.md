# Golar

> Architecture is inspired by [@johnsoncodehk](https://github.com/johnsoncodehk)'s [Volar.js](https://github.com/volarjs/volar.js).

![Demo: LSP and CLI](./demo.gif)

## Plans

* Full Vue support
* Angular
* Svelte
* Astro
* MDX
* Ember
* type aware linting + custom JS plugins?

## Building

```bash
git submodule update --init

cd typescript-go
git am --3way --no-gpg-sign ../patches/*.patch
cd ..

go build -o golar ./typescript-go/cmd/tsgo
```

## License

[MIT](./LICENSE)
