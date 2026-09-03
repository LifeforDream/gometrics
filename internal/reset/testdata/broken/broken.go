package broken

// Синтаксическая ошибка нужна для TestLoadPackages_ReturnsPackageErrors —
// packages.Load должен вернуть её через pkg.Errors, а loadPackages —
// пробросить наверх как error.
func Foo( {
