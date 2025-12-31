package com.example.petclinic.model;

/**
 * Data Transfer Object for PetType entity.
 */
public class PetTypeDto {
    private Long id;
    private String name;

    public PetTypeDto() {}

    public PetTypeDto(Long id, String name) {
        this.id = id;
        this.name = name;
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
}
