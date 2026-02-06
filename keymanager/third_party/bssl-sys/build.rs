// Copyright 2021 The BoringSSL Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use std::collections::HashSet;
use std::env;
use std::path::Path;
use std::path::PathBuf;
use std::process::Command;

// Keep in sync with the list in include/openssl/opensslconf.h
const OSSL_CONF_DEFINES: &[&str] = &[
    "OPENSSL_NO_ASYNC",
    "OPENSSL_NO_BF",
    "OPENSSL_NO_BLAKE2",
    "OPENSSL_NO_BUF_FREELISTS",
    "OPENSSL_NO_CAMELLIA",
    "OPENSSL_NO_CAPIENG",
    "OPENSSL_NO_CAST",
    "OPENSSL_NO_CMS",
    "OPENSSL_NO_COMP",
    "OPENSSL_NO_CT",
    "OPENSSL_NO_DANE",
    "OPENSSL_NO_DEPRECATED",
    "OPENSSL_NO_DGRAM",
    "OPENSSL_NO_DYNAMIC_ENGINE",
    "OPENSSL_NO_EC_NISTP_64_GCC_128",
    "OPENSSL_NO_EC2M",
    "OPENSSL_NO_EGD",
    "OPENSSL_NO_ENGINE",
    "OPENSSL_NO_GMP",
    "OPENSSL_NO_GOST",
    "OPENSSL_NO_HEARTBEATS",
    "OPENSSL_NO_HW",
    "OPENSSL_NO_IDEA",
    "OPENSSL_NO_JPAKE",
    "OPENSSL_NO_KRB5",
    "OPENSSL_NO_MD2",
    "OPENSSL_NO_MDC2",
    "OPENSSL_NO_OCB",
    "OPENSSL_NO_OCSP",
    "OPENSSL_NO_RC2",
    "OPENSSL_NO_RC5",
    "OPENSSL_NO_RFC3779",
    "OPENSSL_NO_RIPEMD",
    "OPENSSL_NO_RMD160",
    "OPENSSL_NO_SCTP",
    "OPENSSL_NO_SEED",
    "OPENSSL_NO_SM2",
    "OPENSSL_NO_SM3",
    "OPENSSL_NO_SM4",
    "OPENSSL_NO_SRP",
    "OPENSSL_NO_SSL_TRACE",
    "OPENSSL_NO_SSL2",
    "OPENSSL_NO_SSL3",
    "OPENSSL_NO_SSL3_METHOD",
    "OPENSSL_NO_STATIC_ENGINE",
    "OPENSSL_NO_STORE",
    "OPENSSL_NO_WHIRLPOOL",
];

fn get_manifest_dir() -> PathBuf {
    PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR not set"))
}

fn is_truthy(value: &str) -> bool {
    matches!(value, "1" | "true" | "TRUE" | "yes" | "on")
}

fn default_bssl_source_dir() -> PathBuf {
    get_manifest_dir().join("../../boringssl")
}

fn get_bssl_source_dir(bssl_build_dir: &Path) -> PathBuf {
    println!("cargo:rerun-if-env-changed=BORINGSSL_SOURCE_DIR");
    if let Some(source_dir) = env::var_os("BORINGSSL_SOURCE_DIR") {
        return PathBuf::from(source_dir);
    }

    let default_source_dir = default_bssl_source_dir();
    if default_source_dir.join("CMakeLists.txt").exists() {
        return default_source_dir;
    }

    if let Some(parent) = bssl_build_dir.parent() {
        if parent.join("CMakeLists.txt").exists() {
            return parent.to_path_buf();
        }
    }

    panic!(
        "Unable to locate BoringSSL source tree. Set BORINGSSL_SOURCE_DIR or use default layout."
    );
}

fn get_bssl_build_dir() -> PathBuf {
    println!("cargo:rerun-if-env-changed=BORINGSSL_BUILD_DIR");
    if let Some(build_dir) = env::var_os("BORINGSSL_BUILD_DIR") {
        return PathBuf::from(build_dir);
    }

    let source_dir = default_bssl_source_dir();
    source_dir.join("build")
}

fn get_cpp_runtime_lib() -> Option<String> {
    println!("cargo:rerun-if-env-changed=BORINGSSL_RUST_CPPLIB");

    if let Ok(cpp_lib) = env::var("BORINGSSL_RUST_CPPLIB") {
        return Some(cpp_lib);
    }

    if env::var_os("CARGO_CFG_UNIX").is_some() {
        match env::var("CARGO_CFG_TARGET_OS").unwrap().as_ref() {
            "macos" => Some("c++".into()),
            _ => Some("stdc++".into()),
        }
    } else {
        None
    }
}

fn static_lib_file_name(target_os: &str, lib_name: &str) -> String {
    if target_os == "windows" {
        format!("{lib_name}.lib")
    } else {
        format!("lib{lib_name}.a")
    }
}

fn find_lib_dirs(bssl_build_dir: &Path, lib_name: &str, target_os: &str) -> Vec<PathBuf> {
    let lib_file = static_lib_file_name(target_os, lib_name);
    let candidate_dirs = [bssl_build_dir.to_path_buf(), bssl_build_dir.join(lib_name)];

    candidate_dirs
        .iter()
        .cloned()
        .filter(|dir| dir.join(&lib_file).exists())
        .collect()
}

fn find_rust_wrapper_dirs(bssl_build_dir: &Path, target_os: &str) -> Vec<PathBuf> {
    let wrapper_file = static_lib_file_name(target_os, "rust_wrapper");
    let candidate_dirs = [
        bssl_build_dir.join("rust/bssl-sys"),
        bssl_build_dir.join("rust/bssl-sys/Release"),
        bssl_build_dir.join("rust/bssl-sys/Debug"),
    ];

    candidate_dirs
        .iter()
        .cloned()
        .filter(|dir| dir.join(&wrapper_file).exists())
        .collect()
}

fn has_required_artifacts(bssl_build_dir: &Path, target: &str, target_os: &str) -> bool {
    let wrapper_rs = bssl_build_dir
        .join("rust/bssl-sys")
        .join(format!("wrapper_{target}.rs"));

    wrapper_rs.exists()
        && !find_lib_dirs(bssl_build_dir, "crypto", target_os).is_empty()
        && !find_lib_dirs(bssl_build_dir, "ssl", target_os).is_empty()
        && !find_rust_wrapper_dirs(bssl_build_dir, target_os).is_empty()
}

fn command_exists(program: &str) -> bool {
    Command::new(program).arg("--version").output().is_ok()
}

fn run_command_fallible(mut command: Command, description: &str) -> Result<(), String> {
    let status = command
        .status()
        .map_err(|err| format!("Failed to {}: {}", description, err))?;

    if !status.success() {
        return Err(format!("Failed to {}. Exit status: {}", description, status));
    }

    Ok(())
}

fn run_command(command: Command, description: &str) {
    if let Err(err) = run_command_fallible(command, description) {
        panic!("{}", err);
    }
}

fn make_configure_command(
    bssl_source_dir: &Path,
    bssl_build_dir: &Path,
    target: &str,
    cache_exists: bool,
) -> Command {
    let mut configure = Command::new("cmake");
    configure
        .arg("-S")
        .arg(bssl_source_dir)
        .arg("-B")
        .arg(bssl_build_dir)
        .arg(format!("-DRUST_BINDINGS={target}"))
        .arg("-DCMAKE_POSITION_INDEPENDENT_CODE=ON")
        .arg("-DBUILD_TESTING=OFF");

    if !cache_exists {
        if let Ok(generator) = env::var("BORINGSSL_CMAKE_GENERATOR") {
            if !generator.is_empty() {
                configure.arg("-G").arg(generator);
            }
        } else if command_exists("ninja") {
            configure.arg("-GNinja");
        }
    }

    configure
}

fn build_boringssl(bssl_source_dir: &Path, bssl_build_dir: &Path, target: &str) {
    if !command_exists("cmake") {
        panic!("cmake is required to build BoringSSL automatically");
    }

    println!("cargo:rerun-if-env-changed=BORINGSSL_CMAKE_GENERATOR");
    let cache_file = bssl_build_dir.join("CMakeCache.txt");
    let cache_exists = cache_file.exists();
    let configure = make_configure_command(bssl_source_dir, bssl_build_dir, target, cache_exists);
    if let Err(err) = run_command_fallible(configure, "configure BoringSSL") {
        if cache_exists {
            println!(
                "cargo:warning={} Recreating BoringSSL build directory and retrying.",
                err
            );
            std::fs::remove_dir_all(bssl_build_dir).unwrap_or_else(|remove_err| {
                panic!(
                    "Failed to remove stale BoringSSL build directory '{}': {}",
                    bssl_build_dir.display(),
                    remove_err
                )
            });

            let configure_retry =
                make_configure_command(bssl_source_dir, bssl_build_dir, target, false);
            run_command(configure_retry, "configure BoringSSL");
        } else {
            panic!("{}", err);
        }
    }

    let mut build = Command::new("cmake");
    build
        .arg("--build")
        .arg(bssl_build_dir)
        .arg("--target")
        .arg("bssl_sys");
    if let Ok(jobs) = env::var("NUM_JOBS") {
        if !jobs.is_empty() {
            build.arg("--parallel").arg(jobs);
        }
    }

    run_command(build, "build BoringSSL");
}

fn ensure_boringssl_ready(bssl_build_dir: &Path, target: &str, target_os: &str) -> PathBuf {
    let bssl_source_dir = get_bssl_source_dir(bssl_build_dir);

    println!("cargo:rerun-if-env-changed=BORINGSSL_RUST_NO_AUTO_BUILD");
    if has_required_artifacts(bssl_build_dir, target, target_os) {
        return bssl_source_dir;
    }

    if let Ok(value) = env::var("BORINGSSL_RUST_NO_AUTO_BUILD") {
        if is_truthy(&value) {
            panic!(
                "Missing BoringSSL build artifacts in '{}' and auto-build is disabled",
                bssl_build_dir.display()
            );
        }
    }

    println!(
        "cargo:warning=Building BoringSSL in '{}' for target '{}'",
        bssl_build_dir.display(),
        target
    );
    build_boringssl(&bssl_source_dir, bssl_build_dir, target);

    if !has_required_artifacts(bssl_build_dir, target, target_os) {
        panic!(
            "BoringSSL build completed but expected artifacts were not found in '{}'",
            bssl_build_dir.display()
        );
    }

    bssl_source_dir
}

fn main() {
    let bssl_build_dir = get_bssl_build_dir();
    let target = env::var("TARGET").expect("TARGET not set");
    let target_os = env::var("CARGO_CFG_TARGET_OS").expect("CARGO_CFG_TARGET_OS not set");

    let bssl_source_dir = ensure_boringssl_ready(&bssl_build_dir, &target, &target_os);
    println!(
        "cargo:rerun-if-changed={}",
        bssl_source_dir.join("CMakeLists.txt").display()
    );
    println!(
        "cargo:rerun-if-changed={}",
        bssl_source_dir.join("rust/bssl-sys/CMakeLists.txt").display()
    );

    let bssl_sys_build_dir = bssl_build_dir.join("rust/bssl-sys");
    let out_dir = env::var("OUT_DIR").expect("OUT_DIR not set");
    let bindgen_out_file = Path::new(&out_dir).join("bindgen.rs");

    // Find the bindgen generated target platform bindings file and put it into
    // OUT_DIR/bindgen.rs.
    let bindgen_source_file = bssl_sys_build_dir.join(format!("wrapper_{target}.rs"));
    let prefix_inc_source_file = bssl_source_dir.join("rust/bssl-sys/boringssl_prefix_symbols_bindgen.rs.in");
    let bindgen_source = std::fs::read_to_string(&bindgen_source_file).unwrap_or_else(|_| {
        panic!(
            "Could not read bindings from '{}'",
            bindgen_source_file.display()
        )
    });
    println!("cargo:rerun-if-changed={}", bindgen_source_file.display());

    let prefix_source = match env::var("BORINGSSL_PREFIX") {
        Ok(prefix) => {
            // Preprocess the file to insert the prefix.
            std::fs::read_to_string(&prefix_inc_source_file)
                .unwrap_or_else(|_| {
                    panic!(
                        "Could not read prefixing data from '{}'",
                        prefix_inc_source_file.display(),
                    )
                })
                .replace("${BORINGSSL_PREFIX}", prefix.as_str())
        }
        Err(env::VarError::NotPresent) => {
            // Just do not append anything.
            "".to_string()
        }
        Err(e) => panic!("failed to read BORINGSSL_PREFIX variable: {}", e),
    };

    std::fs::write(
        &bindgen_out_file,
        format!("{}{}", bindgen_source, prefix_source),
    )
    .unwrap_or_else(|_| {
        panic!(
            "Could not write bindings to '{}'",
            bindgen_out_file.display()
        )
    });

    println!("cargo:rerun-if-changed={}", prefix_inc_source_file.display());
    println!("cargo:rerun-if-env-changed=BORINGSSL_PREFIX");

    let mut link_dirs = HashSet::new();
    link_dirs.extend(find_lib_dirs(&bssl_build_dir, "crypto", &target_os));
    link_dirs.extend(find_lib_dirs(&bssl_build_dir, "ssl", &target_os));
    link_dirs.extend(find_rust_wrapper_dirs(&bssl_build_dir, &target_os));

    for link_dir in link_dirs {
        println!("cargo:rustc-link-search=native={}", link_dir.display());
    }

    // Statically link libraries.
    println!("cargo:rustc-link-lib=static=crypto");
    println!("cargo:rustc-link-lib=static=ssl");
    println!("cargo:rustc-link-lib=static=rust_wrapper");

    if let Some(cpp_lib) = get_cpp_runtime_lib() {
        println!("cargo:rustc-link-lib={}", cpp_lib);
    }

    println!("cargo:conf={}", OSSL_CONF_DEFINES.join(","));
}
