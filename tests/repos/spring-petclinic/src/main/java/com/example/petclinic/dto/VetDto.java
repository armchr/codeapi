package com.example.petclinic.dto;

import jakarta.validation.constraints.NotBlank;
import java.util.Set;

/**
 * Data Transfer Object for Vet entity.
 */
public record VetDto(
    Long id,

    @NotBlank(message = "First name is required")
    String firstName,

    @NotBlank(message = "Last name is required")
    String lastName,

    Set<String> specialties
) {
    /**
     * Creates a VetDto with minimal information.
     */
    public static VetDto of(Long id, String firstName, String lastName) {
        return new VetDto(id, firstName, lastName, Set.of());
    }
}
