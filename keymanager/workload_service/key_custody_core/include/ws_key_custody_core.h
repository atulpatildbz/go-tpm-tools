// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef WS_KEY_CUSTODY_CORE_H_
#define WS_KEY_CUSTODY_CORE_H_

#include <stdint.h>
#include <stddef.h>

// Generates a new Binding keypair.
// key_handle_out: pointer to a 16-byte buffer to receive the key handle (UUID).
// Returns 0 on success, -1 on failure.
int32_t key_manager_generate_binding_keypair(uint8_t *key_handle_out);

// Generates a new KEM keypair associated with a binding public key.
// binding_pk_ptr: pointer to binding public key bytes.
// binding_pk_len: length of binding public key.
// key_handle_out: pointer to a 16-byte buffer to receive the KEM key handle (UUID).
// Returns 0 on success, -1 on failure.
int32_t key_manager_generate_kem_keypair(const uint8_t *binding_pk_ptr,
                                         size_t binding_pk_len,
                                         uint8_t *key_handle_out);

#endif // WS_KEY_CUSTODY_CORE_H_
