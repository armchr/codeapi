package com.example.petclinic.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import java.util.List;

/**
 * Data Transfer Object for Owner entity.
 */
public record OwnerDto(
    Long id,

    @NotBlank(message = "First name is required")
    @Size(min = 2, max = 50)
    String firstName,

    @NotBlank(message = "Last name is required")
    @Size(min = 2, max = 50)
    String lastName,

    String address,
    String city,
    String telephone,

    @Email(message = "Invalid email format")
    String email,

    List<PetDto> pets
) {
    /**
     * Creates an OwnerDto without pets.
     */
    public static OwnerDto of(Long id, String firstName, String lastName, String email) {
        return new OwnerDto(id, firstName, lastName, null, null, null, email, null);
    }
}
