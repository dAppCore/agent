# C++ Build Flow

1. `cmake -B build -DCMAKE_BUILD_TYPE=Release` — configure
2. `cmake --build build -j$(nproc)` — compile
3. `ctest --test-dir build` — run tests
4. `cmake --build build --target install` — install
