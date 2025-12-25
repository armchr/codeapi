package com.example.petclinic.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.time.LocalDate;

/**
 * Data Transfer Object for Pet entity.
 */
public record PetDto(
    Long id,

    @NotBlank(message = "Pet name is required")
    String name,

    @NotNull(message = "Birth date is required")
    LocalDate birthDate,

    String typeName,
    Long ownerId
) {
    /**
     * Creates a PetDto with minimal information.
     */
    public static PetDto of(Long id, String name, LocalDate birthDate) {
        return new PetDto(id, name, birthDate, null, null);
    }
}
