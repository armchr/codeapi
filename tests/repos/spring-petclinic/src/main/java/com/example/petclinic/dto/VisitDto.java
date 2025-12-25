package com.example.petclinic.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.time.LocalDate;

/**
 * Data Transfer Object for Visit entity.
 */
public record VisitDto(
    Long id,

    @NotNull(message = "Visit date is required")
    LocalDate date,

    @NotBlank(message = "Description is required")
    String description,

    Long petId,
    String petName,
    Long vetId,
    String vetName
) {
    /**
     * Creates a VisitDto for creating a new visit.
     */
    public static VisitDto forCreate(LocalDate date, String description, Long petId, Long vetId) {
        return new VisitDto(null, date, description, petId, null, vetId, null);
    }
}
