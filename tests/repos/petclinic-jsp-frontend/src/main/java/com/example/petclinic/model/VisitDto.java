package com.example.petclinic.model;

import java.time.LocalDate;

/**
 * Data Transfer Object for Visit entity.
 */
public class VisitDto {
    private Long id;
    private LocalDate date;
    private String description;
    private Long petId;
    private String petName; // For display purposes

    public VisitDto() {}

    public VisitDto(Long id, LocalDate date, String description, Long petId) {
        this.id = id;
        this.date = date;
        this.description = description;
        this.petId = petId;
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public LocalDate getDate() { return date; }
    public void setDate(LocalDate date) { this.date = date; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public Long getPetId() { return petId; }
    public void setPetId(Long petId) { this.petId = petId; }

    public String getPetName() { return petName; }
    public void setPetName(String petName) { this.petName = petName; }
}
