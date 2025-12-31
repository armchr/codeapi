package com.example.petclinic.model;

import java.time.LocalDate;
import java.time.Period;

/**
 * Data Transfer Object for Pet entity.
 */
public class PetDto {
    private Long id;
    private String name;
    private LocalDate birthDate;
    private String typeName;
    private Long ownerId;
    private String ownerName; // For display purposes

    public PetDto() {}

    public PetDto(Long id, String name, LocalDate birthDate, String typeName, Long ownerId) {
        this.id = id;
        this.name = name;
        this.birthDate = birthDate;
        this.typeName = typeName;
        this.ownerId = ownerId;
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public LocalDate getBirthDate() { return birthDate; }
    public void setBirthDate(LocalDate birthDate) { this.birthDate = birthDate; }

    public String getTypeName() { return typeName; }
    public void setTypeName(String typeName) { this.typeName = typeName; }

    public Long getOwnerId() { return ownerId; }
    public void setOwnerId(Long ownerId) { this.ownerId = ownerId; }

    public String getOwnerName() { return ownerName; }
    public void setOwnerName(String ownerName) { this.ownerName = ownerName; }

    /**
     * Calculate pet's age from birth date.
     */
    public int getAge() {
        if (birthDate == null) return 0;
        return Period.between(birthDate, LocalDate.now()).getYears();
    }
}
