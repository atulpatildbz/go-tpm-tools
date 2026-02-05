//! FFI interface for the WSD key manager operations.

use crate::KeyManager;
use once_cell::sync::Lazy;
use std::slice;
use uuid::Uuid;

static MANAGER: Lazy<KeyManager> = Lazy::new(KeyManager::new);

/// Generates a new Binding keypair.
///
/// # Arguments
/// * `key_handle_out` - Pointer to a 16-byte buffer to receive the key handle.
///
/// # Returns
/// 0 on success, -1 on failure.
///
/// # Safety
/// Assumes `key_handle_out` is a valid pointer to 16 bytes.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn key_manager_generate_binding_keypair(key_handle_out: *mut u8) -> i32 {
    let (handle, _) = MANAGER.generate_binding_keypair();
    unsafe {
        key_handle_out.copy_from_nonoverlapping(handle.as_bytes().as_ptr(), 16);
    }
    0
}

/// Generates a new KEM keypair associated with a binding public key.
///
/// # Arguments
/// * `binding_pk_ptr` - Pointer to binding public key bytes.
/// * `binding_pk_len` - Length of binding public key.
/// * `key_handle_out` - Pointer to a 16-byte buffer to receive the KEM key handle.
///
/// # Returns
/// 0 on success, -1 on failure.
///
/// # Safety
/// Assumes `binding_pk_ptr` and `key_handle_out` are valid.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn key_manager_generate_kem_keypair(
    binding_pk_ptr: *const u8,
    binding_pk_len: usize,
    key_handle_out: *mut u8,
) -> i32 {
    let binding_pk = unsafe { slice::from_raw_parts(binding_pk_ptr, binding_pk_len) };
    let (handle, _) = MANAGER.generate_kem_keypair(binding_pk);
    unsafe {
        key_handle_out.copy_from_nonoverlapping(handle.as_bytes().as_ptr(), 16);
    }
    0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ffi_generate_binding_keypair() {
        let mut handle_buffer = [0u8; 16];

        unsafe {
            let result = key_manager_generate_binding_keypair(handle_buffer.as_mut_ptr());
            assert_eq!(result, 0);
        }

        assert_ne!(handle_buffer, [0u8; 16]);
        let handle = Uuid::from_bytes(handle_buffer);
        assert!(!handle.is_nil());
    }

    #[test]
    fn test_ffi_generate_kem_keypair() {
        let binding_pk = vec![0u8; 32];
        let mut handle_buffer = [0u8; 16];

        unsafe {
            let result = key_manager_generate_kem_keypair(
                binding_pk.as_ptr(),
                binding_pk.len(),
                handle_buffer.as_mut_ptr(),
            );
            assert_eq!(result, 0);
        }

        assert_ne!(handle_buffer, [0u8; 16]);
        let handle = Uuid::from_bytes(handle_buffer);
        assert!(!handle.is_nil());
    }
}
