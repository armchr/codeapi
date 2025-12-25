package com.example.petclinic.controller;

import com.example.petclinic.dto.PetDto;
import com.example.petclinic.service.PetService;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.util.List;
import java.util.concurrent.CompletableFuture;

/**
 * REST controller for Pet operations.
 */
@RestController
@RequestMapping("/api/pets")
public class PetController {

    private final PetService petService;

    public PetController(PetService petService) {
        this.petService = petService;
    }

    /**
     * GET /api/pets - Get all pets.
     */
    @GetMapping
    public ResponseEntity<List<PetDto>> getAllPets() {
        return ResponseEntity.ok(petService.findAll());
    }

    /**
     * GET /api/pets/{id} - Get pet by ID.
     */
    @GetMapping("/{id}")
    public ResponseEntity<PetDto> getPetById(@PathVariable Long id) {
        return ResponseEntity.ok(petService.findById(id));
    }

    /**
     * GET /api/pets/owner/{ownerId} - Get pets by owner ID.
     */
    @GetMapping("/owner/{ownerId}")
    public ResponseEntity<List<PetDto>> getPetsByOwner(@PathVariable Long ownerId) {
        return ResponseEntity.ok(petService.findByOwnerId(ownerId));
    }

    /**
     * GET /api/pets/search?name={name} - Async search pets by name.
     */
    @GetMapping("/search")
    public CompletableFuture<ResponseEntity<List<PetDto>>> searchPets(@RequestParam String name) {
        return petService.searchByNameAsync(name)
            .thenApply(ResponseEntity::ok);
    }

    /**
     * GET /api/pets/stats/type/{typeName} - Get pet count by type.
     */
    @GetMapping("/stats/type/{typeName}")
    public ResponseEntity<Long> countByType(@PathVariable String typeName) {
        return ResponseEntity.ok(petService.countByType(typeName));
    }

    /**
     * POST /api/pets - Create a new pet.
     */
    @PostMapping
    public ResponseEntity<PetDto> createPet(@Valid @RequestBody PetDto petDto) {
        PetDto created = petService.create(petDto);
        return new ResponseEntity<>(created, HttpStatus.CREATED);
    }

    /**
     * PUT /api/pets/{id} - Update an existing pet.
     */
    @PutMapping("/{id}")
    public ResponseEntity<PetDto> updatePet(@PathVariable Long id,
                                             @Valid @RequestBody PetDto petDto) {
        return ResponseEntity.ok(petService.update(id, petDto));
    }

    /**
     * DELETE /api/pets/{id} - Delete a pet.
     */
    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deletePet(@PathVariable Long id) {
        petService.delete(id);
        return ResponseEntity.noContent().build();
    }
}
